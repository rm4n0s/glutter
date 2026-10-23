package androidtool

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/rm4n0s/errors"
)

// DebugKey is the signing key of debug builds: a PKCS#8 private key and an X.509 certificate,
// the two files `apksigner sign --key --cert` takes.
//
// Flutter: no counterpart
type DebugKey struct {
	// Key and Cert are the paths of the files.
	Key, Cert string
}

// EnsureDebugKey returns the debug key in dir, and makes it first when it is not there, with
// Go's crypto packages (SPEC §24.1). It is made once and reused, so that an app updated in
// place keeps its signature. dir is $XDG_CONFIG_HOME/glutter/android-debug by default.
//
// Flutter: no counterpart
func EnsureDebugKey(dir string) (*DebugKey, error) {
	k := &DebugKey{Key: filepath.Join(dir, "key.pk8"), Cert: filepath.Join(dir, "cert.pem")}
	_, errKey := os.Stat(k.Key)
	_, errCert := os.Stat(k.Cert)
	if errKey == nil && errCert == nil {
		return k, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("dir", dir)
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("dir", dir)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("dir", dir)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Glutter Debug", Organization: []string{"Glutter"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(30, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("dir", dir)
	}
	pk8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("dir", dir)
	}
	if err := os.WriteFile(k.Key, pk8, 0o600); err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("path", k.Key)
	}
	if err := os.WriteFile(k.Cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return nil, errors.NewErr(TagAndroidKeyFailed, err).SetMetadata("path", k.Cert)
	}
	return k, nil
}

// TagAndroidKeyFailed is the tag of the error for a debug key that cannot be made or written.
const TagAndroidKeyFailed = "AndroidKeyFailed"
