const c = require("crypto");

function hash_route(r) {
    const header_query_value = r.variables.header_query_value_route;
    const hashed_value = c.createHash('sha256').update(header_query_value).digest('hex');
    return hashed_value;
}

function hash_spec(r) {
    const header_query_value = r.variables.header_query_value_spec;
    const hashed_value = c.createHash('sha256').update(header_query_value).digest('hex');
    return hashed_value;
}



function validate_route(r) {
    const client_name = r.variables['apikey_auth_local_map_route'];
    const header_query_value = r.variables.header_query_value_route;

    if (!header_query_value) {
        r.return(401, "401")
    }
    else if (!client_name) {
        r.return(403, "403")
    }
    else {
        r.return(204, "204");
    }
}

function validate_spec(r) {
    const client_name = r.variables['apikey_auth_local_map_spec'];
    const header_query_value = r.variables.header_query_value_spec;

    if (!header_query_value) {
        r.return(401, "401")
    }
    else if (!client_name) {
        r.return(403, "403")
    }
    else {
        r.return(204, "204");
    }

}

export default { validate_route, validate_spec, hash_route, hash_spec };
