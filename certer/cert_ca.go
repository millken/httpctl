package certer

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const RootName = "rootCA.pem"
const RootKeyName = "rootCA-key.pem"

var tlsClientSkipVerify = &tls.Config{InsecureSkipVerify: true}

var defaultTLSConfig = &tls.Config{
	// InsecureSkipVerify: true,
}
var (
	CaRoot = getCAROOT()
)

func stripPort(s string) string {
	ix := strings.IndexRune(s, ':')
	if ix == -1 {
		return s
	}
	return s[:ix]
}

type CertCA struct {
	storage      *CertStorage
	certPEMBlock []byte
	keyPEMBlock  []byte
	CaCert       *x509.Certificate
	CaKey        crypto.PrivateKey
}

func NewCertCA() *CertCA {
	return &CertCA{
		storage: NewCertStorage(),
	}
}

func (m *CertCA) Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair(m.certPEMBlock, m.keyPEMBlock)
}

func (m *CertCA) GetCertificate(helloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// helloInfo.ServerName ( This contains our Server Name )

	hostname := helloInfo.ServerName

	log.Printf("signing for root %s", hostname)
	genCert := func() (*tls.Certificate, error) {
		cert, err := m.Certificate()
		if err != nil {
			return nil, err
		}
		return signHost(cert, []string{hostname})
	}

	return m.storage.Fetch(hostname, genCert)
}

func (m *CertCA) HostTLSConfig(host string) (*tls.Config, error) {
	var err error
	var cert *tls.Certificate

	hostname := stripPort(host)
	config := defaultTLSConfig.Clone()

	genCert := func() (*tls.Certificate, error) {
		cert, err := m.Certificate()
		if err != nil {
			return nil, err
		}
		return signHost(cert, []string{hostname})
	}

	cert, err = m.storage.Fetch(hostname, genCert)

	if err != nil {
		log.Printf("Cannot sign host certificate with provided CA %s %v", host, err)
		return nil, err
	}

	// config.InsecureSkipVerify = true
	config.ServerName = ""
	config.Certificates = append(config.Certificates, *cert)
	return config, nil
}

// LoadCA will load or create the CA at CAROOT.
func (m *CertCA) LoadCA() error {

	certPEMBlock, err := os.ReadFile(filepath.Join(CaRoot, RootName))
	if err != nil {
		return errors.Wrap(err, "failed to read CA certificate")
	}
	m.certPEMBlock = certPEMBlock
	certDERBlock, _ := pem.Decode(certPEMBlock)
	if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
		return errors.New("failed to read the CA certificate: unexpected content")
	}
	m.CaCert, err = x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse the CA certificate")
	}

	keyPEMBlock, err := os.ReadFile(filepath.Join(CaRoot, RootKeyName))
	if err != nil {
		return errors.Wrap(err, "failed to read the CA key")
	}
	m.keyPEMBlock = keyPEMBlock
	keyDERBlock, _ := pem.Decode(keyPEMBlock)
	if keyDERBlock == nil || keyDERBlock.Type != "PRIVATE KEY" {
		return errors.New("failed to read the CA key: unexpected content")
	}
	m.CaKey, err = x509.ParsePKCS8PrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse the CA key")
	}
	return nil
}

func getCAROOT() string {
	if env := os.Getenv("CAROOT"); env != "" {
		return env
	}

	var dir string
	switch {
	case runtime.GOOS == "windows":
		dir = os.Getenv("LocalAppData")
	case os.Getenv("XDG_DATA_HOME") != "":
		dir = os.Getenv("XDG_DATA_HOME")
	case runtime.GOOS == "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return ""
		}
		dir = filepath.Join(dir, "Library", "Application Support")
	default: // Unix
		dir = os.Getenv("HOME")
		if dir == "" {
			return ""
		}
		dir = filepath.Join(dir, ".local", "share")
	}
	return filepath.Join(dir, "foddler")
}

func signHost(ca tls.Certificate, hosts []string) (cert *tls.Certificate, err error) {
	var x509ca *x509.Certificate

	// Use the provided ca and not the global GoproxyCa for certificate generation.
	if x509ca, err = x509.ParseCertificate(ca.Certificate[0]); err != nil {
		return
	}
	start := time.Unix(0, 0)
	end, err := time.Parse("2006-01-02", "2049-12-31")
	if err != nil {
		panic(err)
	}

	serial := big.NewInt(rand.Int63())
	template := x509.Certificate{
		// TODO(elazar): instead of this ugly hack, just encode the certificate and hash the binary form.
		SerialNumber: serial,
		Issuer:       x509ca.Subject,
		Subject: pkix.Name{
			Organization: []string{"Foddler MITM proxy Inc"},
		},
		NotBefore: start,
		NotAfter:  end,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
			template.Subject.CommonName = h
		}
	}

	var csprng CounterEncryptorRand
	if csprng, err = NewCounterEncryptorRandFromKey(ca.PrivateKey, nil); err != nil {
		return
	}

	var certpriv crypto.Signer
	switch ca.PrivateKey.(type) {
	case *rsa.PrivateKey:
		if certpriv, err = rsa.GenerateKey(&csprng, 2048); err != nil {
			return
		}
	case *ecdsa.PrivateKey:
		if certpriv, err = ecdsa.GenerateKey(elliptic.P256(), &csprng); err != nil {
			return
		}
	default:
		err = fmt.Errorf("unsupported key type %T", ca.PrivateKey)
	}

	var derBytes []byte
	if derBytes, err = x509.CreateCertificate(&csprng, &template, x509ca, certpriv.Public(), ca.PrivateKey); err != nil {
		return
	}
	return &tls.Certificate{
		Certificate: [][]byte{derBytes, ca.Certificate[0]},
		PrivateKey:  certpriv,
	}, nil
}
