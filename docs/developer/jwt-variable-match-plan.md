# Plan: Expand VirtualServer & VirtualServerRoute Match Routing to Support `$jwt_` Variables (#11175)

## Status: Draft

## Summary

**Goal:** Expand `VirtualServer` (VS) and `VirtualServerRoute` (VSR) conditional routing (`matches[].conditions[].variable`) and NGINX Plus variable lists in NGINX Ingress Controller (NIC) to support NGINX Plus `ngx_http_auth_jwt_module` variables—including built-in top-level claims (`$jwt_claim_<name>`), custom/nested claims extracted via `auth_jwt_claim_set` using the `$jwt_custom_<path>` prefix with `.` as the nested path delimiter (e.g., `$jwt_custom_user_details.level`), JOSE headers (`$jwt_header_<name>`), and payload (`$jwt_payload`)—across both server-level and route/subroute-level `JWTAuth` policies.

**References:**
- Issue: [#11175](https://github.com/nginx/kubernetes-ingress/issues/11175) (parent: [#6829](https://github.com/nginx/kubernetes-ingress/issues/6829), sibling: [#11174](https://github.com/nginx/kubernetes-ingress/issues/11174))
- NIC Documentation: [VirtualServer and VirtualServerRoute Resources — Condition](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#condition)
- NGINX Plus Documentation: [`ngx_http_auth_jwt_module` (`auth_jwt_claim_set` & Embedded Variables)](https://nginx.org/en/docs/http/ngx_http_auth_jwt_module.html#variables)

---

## API Design

No new CRD struct fields are required on `Condition` (`pkg/apis/configuration/v1/types.go`). Instead, expand the allowed NGINX variables in `routes[].matches[].conditions[].variable` (for `VirtualServer`) and `subroutes[].matches[].conditions[].variable` (for `VirtualServerRoute`) when running with **NGINX Plus** (`isPlus == true`).

Because NGINX does not allow `.` in variable names and NGINX Plus's built-in `$jwt_claim_<name>` variable only resolves top-level scalar claims, NIC separates top-level built-in claim variables from `auth_jwt_claim_set`-backed claims by prefix:
- `$jwt_claim_<name>` is kept strictly for NGINX Plus's built-in top-level claim variables (`[a-zA-Z0-9_]+`, passed directly to NGINX).
- `$jwt_custom_<path>` is used in `condition.variable` (with `.` as the nested claim path delimiter, e.g., `$jwt_custom_user_details.level`, consistent with `RateLimit` `condition.jwt.claim` and `JWTAuth` claim validation in [#11174](https://github.com/nginx/kubernetes-ingress/issues/11174)) to instruct NIC to extract the claim via `auth_jwt_claim_set` in the `http` context and reference the generated NGINX variable in the condition `map`.

For a JWT payload containing both top-level and nested claims:

```json
{
  "user_details": {
    "level": "Premium"
  },
  "sub": "client1",
  "name": "John Doe"
}
```

Users can match on nested claims (`$jwt_custom_user_details.level`), top-level claims (`$jwt_claim_sub`), JOSE headers (`$jwt_header_alg`), and `$jwt_payload`:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: cafe
spec:
  host: cafe.example.com
  upstreams:
    - name: premium-backend
      service: premium-svc
      port: 80
    - name: regular-backend
      service: regular-svc
      port: 80
  routes:
    - path: /coffee
      policies:
        - name: jwt-policy
      matches:
        - conditions:
            - variable: $jwt_custom_user_details.level
              value: Premium
            - variable: $jwt_claim_sub
              value: client1
            - variable: $jwt_header_alg
              value: RS256
          action:
            proxy:
              upstream: premium-backend
              requestHeaders:
                set:
                  - name: X-JWT-Sub
                    value: ${jwt_claim_sub}
                  - name: X-JWT-Payload
                    value: ${jwt_payload}
      action:
        pass: regular-backend
```

### Supported `$jwt_` Variables Specification

| Variable Pattern | Contexts Supported | NGINX Plus Only | Validation / Constraints | Description |
|---|---|---|---|---|
| `$jwt_claim_<name>` | • `matches[].conditions[].variable`<br>• `action.proxy.requestHeaders.set[].value` (`${jwt_claim_<name>}`) | Yes | `<name>` must be non-empty and consist of alphanumeric characters or `_` (`isValidSpecialHeaderLikeVariable`, no `.` allowed) | Evaluates to the value of a top-level JWT claim (e.g., `$jwt_claim_sub`) using NGINX Plus's built-in variable directly. |
| `$jwt_custom_<path>` | • `matches[].conditions[].variable` | Yes | `<path>` consists of one or more `.`-separated segments (`<seg1>.<seg2>...`); each segment must be non-empty and pass `validation.IsQualifiedName` / `isValidSpecialHeaderLikeVariable` | Triggers `auth_jwt_claim_set` generation in the `http` context (e.g., `$jwt_custom_user_details.level` -> `auth_jwt_claim_set $jwt_<ns>_<name>_vs_user_details_level user_details level;`) and uses the synthesized variable in the condition `map`. |
| `$jwt_header_<name>` | • `matches[].conditions[].variable`<br>• `action.proxy.requestHeaders.set[].value` (`${jwt_header_<name>}`) | Yes | `<name>` must be non-empty and consist of alphanumeric characters or `_` (`isValidSpecialHeaderLikeVariable`, no `.` allowed) | Evaluates to the value of the specified JOSE header parameter (e.g., `alg`, `kid`, `typ`). |
| `$jwt_payload` | • `matches[].conditions[].variable`<br>• `action.proxy.requestHeaders.set[].value` (`${jwt_payload}`) | Yes | Exact variable name `jwt_payload` (no suffix allowed) | Evaluates to the decrypted top-level payload of a JWS or JWE token. |

---

## NGINX Plus Configuration Design

Using NGINX Plus `ngx_http_auth_jwt_module` and `ngx_http_map_module`:

1. **Custom / Nested Claim Extraction via `auth_jwt_claim_set` (`http` context)**:
   - NGINX variable names cannot contain `.`, and NGINX Plus's built-in `$jwt_claim_<name>` variable only resolves top-level scalar claims.
   - When a match condition variable starts with `$jwt_custom_` (e.g., `$jwt_custom_user_details.level`), NIC strips the `$jwt_custom_` prefix to obtain the claim path (`user_details.level`) and generates an `auth_jwt_claim_set` entry using `generateAuthJwtClaimSetVariable` and `generateAuthJwtClaimSetClaim` (`internal/configs/policy.go`):
     ```nginx
     auth_jwt_claim_set $jwt_default_cafe_vs_user_details_level user_details level;
     ```
   - Because it reuses the same variable naming convention (`$jwt_<vs_ns>_<vs_name>_vs_<claim_segments>`) as `RateLimit` (`JWTCondition`) and `JWTAuth` claim validation ([#11174](https://github.com/nginx/kubernetes-ingress/issues/11174)), `removeDuplicateAuthJWTClaimSets` in `internal/configs/virtualserver.go` automatically deduplicates `auth_jwt_claim_set` directives when the same claim is referenced by both a `Policy` and a `VirtualServer` / `VirtualServerRoute` `match` condition.
2. **Condition & Match Map Generation (`http` context)**:
   - For built-in variables (`$jwt_claim_sub`, `$jwt_header_alg`, `$jwt_payload`), `generateMatchesConfig` uses `condition.Variable` directly as the `map` source.
   - For `$jwt_custom_<path>` variables (`$jwt_custom_user_details.level`), `generateMatchesConfig` uses the synthesized `auth_jwt_claim_set` variable (`$jwt_default_cafe_vs_user_details_level`) as the `map` source:
     ```nginx
     auth_jwt_claim_set $jwt_default_cafe_vs_user_details_level user_details level;

     map $jwt_default_cafe_vs_user_details_level $vs_default_cafe_matches_0_match_0_cond_0 {
         "Premium" 1;
         default 0;
     }

     map $jwt_claim_sub $vs_default_cafe_matches_0_match_0_cond_1 {
         "client1" 1;
         default 0;
     }

     map $jwt_header_alg $vs_default_cafe_matches_0_match_0_cond_2 {
         "RS256" 1;
         default 0;
     }

     map $vs_default_cafe_matches_0_match_0_cond_0$vs_default_cafe_matches_0_match_0_cond_1$vs_default_cafe_matches_0_match_0_cond_2 $vs_default_cafe_matches_0 {
         ~^111$ /internal_location_matches_0_match_0;
         default /internal_location_matches_0_default;
     }
     ```
3. **Entry Redirect Location & Route-Level `JWTAuth` Evaluation (`server` context)**:
   - For a route or subroute with `matches`, NIC generates an `InternalRedirectLocation` (`location /coffee { rewrite ^ $vs_default_cafe_matches_0 last; }`).
   - Because `rewrite` executes in `NGX_HTTP_REWRITE_PHASE` (before redirecting to `/internal_location_matches_...`), NGINX evaluates `$vs_default_cafe_matches_0`—and therefore `auth_jwt_claim_set` variables, `$jwt_claim_*`, `$jwt_header_*`, and `$jwt_payload`—while the request is still in `location /coffee`.
   - In `ngx_http_auth_jwt_module`, the variable getters inspect the current location's `auth_jwt` configuration to locate and decode the token.
   - When `JWTAuth` is attached at the **server level** (`spec.policies`), `location /coffee` inherits `auth_jwt` from `server`.
   - When `JWTAuth` is attached at the **route or subroute level** (`routes[].policies` or `subroutes[].policies`), `InternalRedirectLocation` must also render the route's `JWTAuth` directives (`auth_jwt`, `auth_jwt_key_file`, `auth_jwt_key_cache`, `auth_jwt_key_request`) so JWT variables (both embedded and `auth_jwt_claim_set`) are populated during `rewrite` before jumping to the internal match location:
     ```nginx
     location /coffee {
         auth_jwt "Cafe App" token=$http_token;
         auth_jwt_key_file /etc/nginx/secrets/default-jwt-secret;
         client_max_body_size 0;
         rewrite ^ $vs_default_cafe_matches_0 last;
     }
     ```
4. **Internal Match Target Locations (`server` context)**:
   - Each internal location (`location ~ ^/internal_location_matches_0_match_0$`) continues to receive `routePoliciesCfg` via `addPoliciesCfgToLocations`, enforcing `auth_jwt` signature/expiration verification (and `auth_jwt_require` from [#11174](https://github.com/nginx/kubernetes-ingress/issues/11174)) in `NGX_HTTP_ACCESS_PHASE` and proxying to the matched upstream.

---

## Implementation Phases

### Phase 1: Examples & User Contract

1. **Create / Update Example Manifests**:
   - Update `examples/custom-resources/jwt/` (or add a dedicated routing example) and `docs/content/configuration/virtualserver-and-virtualserverroute-resources.md` to document `$jwt_claim_<name>`, `$jwt_custom_<path>` (for nested/custom claims such as `$jwt_custom_user_details.level`), `$jwt_header_<name>`, and `$jwt_payload` (NGINX Plus only) in `condition.variable` and `action.proxy.requestHeaders`.

### Phase 2: Validation & CRD Documentation

1. **Update `pkg/apis/configuration/validation/common.go`**:
   - Extend `parseSpecialVariable` and `validateSpecialVariable`:
     - Support `jwt_payload` as an exact NGINX Plus special variable (returning `field.Forbidden(fieldPath, "is only supported in NGINX Plus")` when `!isPlus`, and rejecting any trailing suffix such as `jwt_payload_foo`).
     - Support `jwt_custom` as an NGINX Plus special variable (`jwt_custom_<path>`) where `<path>` is split on `.` and each segment is validated with `isValidSpecialHeaderLikeVariable(segment)` (or `validation.IsQualifiedName` matching `validateRateLimitJWTCondition` in `pkg/apis/configuration/validation/policy.go`), while keeping `jwt_claim` and `jwt_header` strictly limited to `[a-zA-Z0-9_]+` without `.`.
2. **Update `pkg/apis/configuration/validation/virtualserver.go` & `virtualserver_test.go`**:
   - Add `"jwt_payload"` to `actionProxyHeaderSpecialVariables` (`[]string{"arg_", "http_", "cookie_", "jwt_claim_", "jwt_header_", "jwt_payload"}`).
   - Define `conditionSpecialVariableNames = []string{"jwt_claim_", "jwt_custom_", "jwt_header_", "jwt_payload"}`.
   - Thread `vsv.isPlus` through `validateMatch` -> `validateCondition` -> `validateVariableName`.
   - In `validateVariableName`, strip the leading `$` and call `validateSpecialVariable(nVar, fieldPath, vsv.isPlus)` when `nVar` matches a prefix in `conditionSpecialVariableNames`; otherwise validate against `validVariableNames`.
   - Add positive and negative table-driven unit tests in `pkg/apis/configuration/validation/virtualserver_test.go` covering:
     - Valid `$jwt_claim_sub`, `$jwt_custom_user_details.level`, `$jwt_header_alg`, and `$jwt_payload` in `matches[].conditions[].variable` when `isPlus: true`.
     - Valid `${jwt_claim_sub}`, `${jwt_header_alg}`, and `${jwt_payload}` in `action.proxy.requestHeaders` when `isPlus: true`.
     - Rejection with `Forbidden` when `isPlus: false` (NGINX OSS).
     - Rejection of malformed names (`$jwt_claim_`, `$jwt_claim_user_details.level`, `$jwt_custom_`, `$jwt_custom_.level`, `$jwt_custom_user_details.`, `$jwt_custom_user_details..level`, `$jwt_header_foo.bar`, `$jwt_payload_extra`, and dangerous characters).
3. **Update CRD Documentation**:
   - Update `pkg/apis/configuration/v1/types.go` comments on `Condition.Variable` if applicable and run `make update-crds` to refresh `docs/crd/k8s.nginx.org_virtualservers.md` and `docs/crd/k8s.nginx.org_virtualserverroutes.md`.

### Phase 3: NGINX Plus Configuration Generation

1. **Update Template Structs in `internal/configs/version2/http.go`**:
   - Add `JWTAuth *JWTAuth` to `InternalRedirectLocation`.
2. **Update Match Config Generation in `internal/configs/virtualserver.go`**:
   - Add `AuthJWTClaimSets []version2.AuthJWTClaimSet` to `routingCfg`.
   - In `generateMatchesConfig`:
     - When `strings.HasPrefix(c.Variable, "$jwt_custom_")`:
       - Extract `claim := strings.TrimPrefix(c.Variable, "$jwt_custom_")`.
       - Generate `version2.AuthJWTClaimSet{Variable: generateAuthJwtClaimSetVariable(claim, ownerDetails), Claim: generateAuthJwtClaimSetClaim(claim)}` and append it to `routingCfg.AuthJWTClaimSets`.
       - Use the generated `AuthJWTClaimSet.Variable` (e.g., `$jwt_default_cafe_vs_user_details_level`) as the `Source` for the condition `version2.Map`.
     - Otherwise, use `getNameForSourceForMatchesRouteMapFromCondition(c)` directly as the `Source` (which returns `c.Variable` for `$jwt_claim_*`, `$jwt_header_*`, `$jwt_payload`, and standard NGINX variables).
   - In `generateRouteConfig` and `generateSubRoutesConfig` (when `len(r.Matches) > 0`):
     - Append `cfg.AuthJWTClaimSets` to `authJWTClaimSets` (which is deduplicated by `removeDuplicateAuthJWTClaimSets(authJWTClaimSets)`).
     - Set `cfg.InternalRedirectLocation.JWTAuth = routePoliciesCfg.JWTAuth.Auth` (for both `matches` and `splits` routes).
   - Add unit tests in `internal/configs/virtualserver_test.go` verifying:
     - Top-level `$jwt_claim_sub`, `$jwt_header_alg`, and `$jwt_payload` map sources.
     - `$jwt_custom_user_details.level` `AuthJWTClaimSet` generation, deduplication with policy-level `AuthJWTClaimSets`, and map source resolution.
     - `InternalRedirectLocation.JWTAuth` population for route-level and subroute-level `JWTAuth` policies.

### Phase 4: Template Rendering & Snapshots

1. **Update `internal/configs/version2/nginx-plus.virtualserver.tmpl`**:
   - Inside `{{- range $l := $s.InternalRedirectLocations }}`, render the `{{- with $l.JWTAuth }}` block (`auth_jwt`, `auth_jwt_key_file`, `auth_jwt_key_cache`, `auth_jwt_key_request`) before `rewrite ^ {{ $l.Destination }} last;`.
   - Keep `internal/configs/version2/nginx.virtualserver.tmpl` (OSS) unchanged so Plus-only `auth_jwt` directives never appear in OSS templates.
2. **Update Snapshot Tests**:
   - Add/update test fixtures in `internal/configs/version2/http_test.go` and `internal/configs/virtualserver_test.go` exercising `$jwt_claim_*`, `$jwt_custom_user_details.level`, `$jwt_header_*`, and `$jwt_payload` match conditions with route-level `JWTAuth`.
   - Run `make test-update-snaps` and `make test` to verify non-empty golden snapshot diffs in `__snapshots__/`.

### Phase 5: End-to-End Integration Tests

1. **Python Integration Tests**:
   - Extend `tests/suite/test_jwt_policies.py` and `tests/suite/test_jwt_policies_vsr.py` (with test manifests in `tests/data/jwt-policy/`) to test conditional routing on top-level claims (`$jwt_claim_sub`), nested claims (`$jwt_custom_user_details.level`), JOSE headers (`$jwt_header_alg`), and `$jwt_payload` across both spec-level (`spec.policies`) and route/subroute-level (`routes[].policies` / `subroutes[].policies`) `JWTAuth` policies.

---

## PR / Commit Stack Structure

To keep reviews focused and incremental, the change can be structured as a 6-part commit / PR stack:

```mermaid
flowchart LR
    C1["1. Design & Test Plan"] --> C2["2. Examples & User Docs"]
    C2 --> C3["3. Validation & Unit Tests"]
    C3 --> C4["4. Config Generation Logic"]
    C4 --> C5["5. Templates & Snapshots"]
    C5 --> C6["6. E2E Integration Tests"]
```

| Order | Scope | Purpose for Reviewers | Files Included |
| :---: | --- | --- | --- |
| **1** | **Design & Test Plan** | Align on supported `$jwt_` variables (top-level `$jwt_claim_<name>`, nested/custom `$jwt_custom_<path>` via `auth_jwt_claim_set`, `$jwt_header_<name>`, `$jwt_payload`), NGINX Plus gating, route-level `InternalRedirectLocation` `auth_jwt` evaluation during `rewrite`, and test strategy. | • [`docs/developer/jwt-variable-match-plan.md`](jwt-variable-match-plan.md) |
| **2** | **Examples & User Docs** | Demonstrate top-level `$jwt_claim_*`, nested `$jwt_custom_*`, and `$jwt_header_*` match routing in `VirtualServer` / `VirtualServerRoute` examples and user documentation. | • [`examples/custom-resources/jwt/virtual-server.yaml`](../../examples/custom-resources/jwt/virtual-server.yaml)<br>• [`examples/custom-resources/jwt/README.md`](../../examples/custom-resources/jwt/README.md)<br>• [`docs/content/configuration/virtualserver-and-virtualserverroute-resources.md`](../content/configuration/virtualserver-and-virtualserverroute-resources.md) |
| **3** | **Validation & Unit Tests** | Review `isPlus`-aware validation for `$jwt_claim_<name>`, `$jwt_custom_<path>`, `$jwt_header_<name>`, and `$jwt_payload` in `matches[].conditions[].variable` and `action.proxy.requestHeaders`, plus positive/negative unit tests. | • [`pkg/apis/configuration/validation/common.go`](../../pkg/apis/configuration/validation/common.go)<br>• [`pkg/apis/configuration/validation/virtualserver.go`](../../pkg/apis/configuration/validation/virtualserver.go)<br>• [`pkg/apis/configuration/validation/virtualserver_test.go`](../../pkg/apis/configuration/validation/virtualserver_test.go) |
| **4** | **Config Generation Logic** | Review `auth_jwt_claim_set` generation & deduplication for `$jwt_custom_<path>` claims in `matches`, `InternalRedirectLocation.JWTAuth` propagation for route/subroute-level `JWTAuth` policies, and unit tests. | • [`internal/configs/version2/http.go`](../../internal/configs/version2/http.go)<br>• [`internal/configs/virtualserver.go`](../../internal/configs/virtualserver.go)<br>• [`internal/configs/virtualserver_test.go`](../../internal/configs/virtualserver_test.go) |
| **5** | **Template Rendering & Snapshots** | Review NGINX Plus `InternalRedirectLocations` `auth_jwt` template rendering and golden snapshot updates (including `auth_jwt_claim_set` + `map` for `$jwt_custom_<path>` claims). | • [`internal/configs/version2/nginx-plus.virtualserver.tmpl`](../../internal/configs/version2/nginx-plus.virtualserver.tmpl)<br>• [`internal/configs/version2/http_test.go`](../../internal/configs/version2/http_test.go)<br>• [`internal/configs/version2/__snapshots__/http_test.snap`](../../internal/configs/version2/__snapshots__/http_test.snap)<br>• [`internal/configs/__snapshots__/virtualserver_test.snap`](../../internal/configs/__snapshots__/virtualserver_test.snap) |
| **6** | **End-to-End Tests** | Verify full runtime behavior in pytest for VS and VSR `matches` routing using top-level `$jwt_claim_sub`, nested `$jwt_custom_user_details.level`, `$jwt_header_*`, and `$jwt_payload` with both server-level and route/subroute-level `JWTAuth` policies. | • [`tests/suite/test_jwt_policies.py`](../../tests/suite/test_jwt_policies.py)<br>• [`tests/suite/test_jwt_policies_vsr.py`](../../tests/suite/test_jwt_policies_vsr.py)<br>• [`tests/data/jwt-policy/`](../../tests/data/jwt-policy/) |
