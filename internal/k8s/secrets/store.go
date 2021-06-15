package secrets

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretReference holds a reference to a secret stored on the file system.
type SecretReference struct {
	Secret *api_v1.Secret
	Path   string
	Error  error
}

// SecretFileManager manages secrets on the file system.
type SecretFileManager interface {
	AddOrUpdateSecret(secret *api_v1.Secret) string
	DeleteSecret(key string)
}

// SecretStore stores secrets that the Ingress Controller uses.
type SecretStore interface {
	AddOrUpdateSecret(secret *api_v1.Secret)
	DeleteSecret(key string)
	GetSecret(key string) *SecretReference
}

// LocalSecretStore implements SecretStore interface.
// It validates the secrets and manages them on the file system (via SecretFileManager).
type LocalSecretStore struct {
	secrets map[string]*SecretReference
	manager SecretFileManager
}

// NewLocalSecretStore creates a new LocalSecretStore.
func NewLocalSecretStore(manager SecretFileManager) *LocalSecretStore {
	return &LocalSecretStore{
		secrets: make(map[string]*SecretReference),
		manager: manager,
	}
}

// AddOrUpdateSecret adds or updates a secret.
// The secret will only be updated on the file system if it is valid and if it is already on the file system.
// If the secret becomes invalid, it will be removed from the filesystem.
func (s *LocalSecretStore) AddOrUpdateSecret(secret *api_v1.Secret) {
	secretRef, exists := s.secrets[getResourceKey(&secret.ObjectMeta)]
	if !exists {
		secretRef = &SecretReference{Secret: secret}
	} else {
		secretRef.Secret = secret
	}

	secretRef.Error = ValidateSecret(secret)

	if secretRef.Path != "" {
		if secretRef.Error != nil {
			s.manager.DeleteSecret(getResourceKey(&secret.ObjectMeta))
			secretRef.Path = ""
		} else {
			secretRef.Path = s.manager.AddOrUpdateSecret(secret)
		}
	}

	s.secrets[getResourceKey(&secret.ObjectMeta)] = secretRef
}

// DeleteSecret deletes a secret.
func (s *LocalSecretStore) DeleteSecret(key string) {
	storedSecret, exists := s.secrets[key]
	if !exists {
		return
	}

	delete(s.secrets, key)

	if storedSecret.Path == "" {
		return
	}

	s.manager.DeleteSecret(key)
}

// GetSecret returns a SecretReference.
// If the secret doesn't exist, is of an unsupported type, or invalid, the Error field will include an error.
// If the secret is valid but isn't present on the file system, the secret will be written to the file system.
func (s *LocalSecretStore) GetSecret(key string) *SecretReference {
	secretRef, exists := s.secrets[key]
	if !exists {
		return &SecretReference{
			Error: fmt.Errorf("secret doesn't exist or of an unsupported type"),
		}
	}

	if secretRef.Error == nil && secretRef.Path == "" {
		secretRef.Path = s.manager.AddOrUpdateSecret(secretRef.Secret)
	}

	return secretRef
}

func getResourceKey(meta *metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

// FakeSecretStore is a fake implementation of SecretStore.
type FakeSecretStore struct {
	secrets map[string]*SecretReference
}

// NewFakeSecretsStore creates a new FakeSecretStore.
func NewFakeSecretsStore(secrets map[string]*SecretReference) *FakeSecretStore {
	return &FakeSecretStore{
		secrets: secrets,
	}
}

// NewEmptyFakeSecretsStore creates a new empty FakeSecretStore.
func NewEmptyFakeSecretsStore() *FakeSecretStore {
	return &FakeSecretStore{
		secrets: make(map[string]*SecretReference),
	}
}

// AddOrUpdateSecret is a fake implementation of AddOrUpdateSecret.
func (s *FakeSecretStore) AddOrUpdateSecret(secret *api_v1.Secret) {
	secretRef, exists := s.secrets[getResourceKey(&secret.ObjectMeta)]
	if !exists {
		secretRef = &SecretReference{Secret: secret}
	} else {
		secretRef.Secret = secret
	}
	s.secrets[getResourceKey(&secret.ObjectMeta)] = secretRef
}

// DeleteSecret is a fake implementation of DeleteSecret.
func (s *FakeSecretStore) DeleteSecret(key string) {
}

// GetSecret is a fake implementation of GetSecret.
func (s *FakeSecretStore) GetSecret(key string) *SecretReference {
	secretRef, exists := s.secrets[key]
	if !exists {
		return &SecretReference{
			Error: fmt.Errorf("secret doesn't exist"),
		}
	}

	return secretRef
}
