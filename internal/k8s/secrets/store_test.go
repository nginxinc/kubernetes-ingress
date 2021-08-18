package secrets

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeSecretFileManager struct {
	AddedOrUpdatedSecret *api_v1.Secret
	DeletedSecret        string
}

func (m *fakeSecretFileManager) AddOrUpdateSecret(secret *api_v1.Secret) string {
	m.AddedOrUpdatedSecret = secret
	return "testpath"
}

func (m *fakeSecretFileManager) DeleteSecret(key string) {
	m.DeletedSecret = key
}

func (m *fakeSecretFileManager) Reset() {
	m.AddedOrUpdatedSecret = nil
	m.DeletedSecret = ""
}

var (
	validSecret = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": validCert,
			"tls.key": validKey,
		},
	}
	invalidSecret = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": invalidCert,
			"tls.key": validKey,
		},
	}
)

func errorComparer(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		return errors.Is(e1, e2)
	}

	return e1.Error() == e2.Error()
}

func TestAddOrUpdateSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}

	store := NewLocalSecretStore(manager)

	// Add the valid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef := &SecretReference{
		Secret: validSecret,
		Path:   "testpath",
		Error:  nil,
	}
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretRef := store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Make the secret invalid

	expectedManager = &fakeSecretFileManager{
		DeletedSecret: "default/tls-secret",
	}

	manager.Reset()
	store.AddOrUpdateSecret(invalidSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef = &SecretReference{
		Secret: invalidSecret,
		Path:   "",
		Error:  errors.New("Failed to validate TLS cert and key: x509: malformed certificate"),
	}
	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	secretRef = store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Restore the valid secret

	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef = &SecretReference{
		Secret: validSecret,
		Path:   "testpath",
		Error:  nil,
	}
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretRef = store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Update the secret

	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	// for the test, it is ok to use the same version
	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef = &SecretReference{
		Secret: validSecret,
		Path:   "testpath",
		Error:  nil,
	}
	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	secretRef = store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretNonExisting(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	expectedManager := &fakeSecretFileManager{}

	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretValidSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	// Add the valid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(validSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef := &SecretReference{
		Secret: validSecret,
		Path:   "testpath",
		Error:  nil,
	}
	expectedManager = &fakeSecretFileManager{
		AddedOrUpdatedSecret: validSecret,
	}

	manager.Reset()
	secretRef := store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete the secret

	expectedManager = &fakeSecretFileManager{
		DeletedSecret: "default/tls-secret",
	}

	manager.Reset()
	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Get the secret

	expectedSecretRef = &SecretReference{
		Error: errors.New("secret doesn't exist or of an unsupported type"),
	}
	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	secretRef = store.GetSecret("default/tls-secret")

	if diff := cmp.Diff(expectedSecretRef, secretRef, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("GetSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestDeleteSecretInvalidSecret(t *testing.T) {
	manager := &fakeSecretFileManager{}
	store := NewLocalSecretStore(manager)

	// Add invalid secret

	expectedManager := &fakeSecretFileManager{}

	store.AddOrUpdateSecret(invalidSecret)

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("AddOrUpdateSecret() returned unexpected result (-want +got):\n%s", diff)
	}

	// Delete invalid secret

	expectedManager = &fakeSecretFileManager{}

	manager.Reset()
	store.DeleteSecret("default/tls-secret")

	if diff := cmp.Diff(expectedManager, manager); diff != "" {
		t.Errorf("DeleteSecret() returned unexpected result (-want +got):\n%s", diff)
	}
}
