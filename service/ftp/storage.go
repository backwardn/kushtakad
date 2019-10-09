// Copyright 2016-2019 DutchSec (https://dutchsec.com/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ftp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/kushtaka/kushtakad/storage"
)

func getStorage() (*ftpStorage, error) {
	s, err := storage.Namespace("ftp")
	if err != nil {
		return nil, err
	}
	return &ftpStorage{
		s,
	}, nil
}

type ftpStorage struct {
	storage.Storage
}

func (s *ftpStorage) FileSystem() (base, serviceroot string, err error) {
	b, err := s.Get("base")
	if err != nil {
		return "", "", err
	}
	base = string(b)

	sr, err := s.Get("fs_root")
	if err != nil {
		return "", "", err
	}
	serviceroot = string(sr)

	return base, serviceroot, nil
}

//Returns a TLS Certificate
func (s *ftpStorage) Certificate() (*tls.Certificate, error) {
	var errOut, errIn error
	var pemkey, pemcert []byte

	keyname := "pemkey"
	certname := "pemcert"

	pemkey, errOut = s.Get(keyname)
	if errOut != nil || len(pemkey) == 0 {
		pemkey, errIn = generateKey()
		log.Debugf("pemkey %s", pemkey)
		if errIn != nil {
			log.Errorf("generateKey %v", errIn)
			return nil, errIn
		}

		errIn = s.Set(keyname, pemkey)
		if errIn != nil {
			log.Errorf("generateKey Set() %v", errIn)
			log.Errorf("Could not persist %s: %s", keyname, errIn.Error())
		}
	}

	pemcert, errOut = s.Get(certname)
	if errOut != nil || len(pemcert) == 0 {
		pemcert, errIn = generateCert(pemkey)
		log.Debugf("pemcert %s", pemcert)
		if errIn != nil {
			log.Errorf("generateCert %v", errIn)
			return nil, errIn
		}

		errIn = s.Set(certname, pemcert)
		if errIn != nil {
			log.Errorf("generateCert Set() %v", errIn)
			log.Errorf("Could not persist %s: %s", certname, errIn.Error())
		}
	}

	log.Debugf("key %s cert %s", pemkey, pemkey)
	tlscert, err := tls.X509KeyPair(pemcert, pemkey)
	if err != nil {
		log.Errorf("tls.X509KeyPair() %v", err)
		return nil, err
	}

	return &tlscert, nil
}

//Returns a PEM encoded RSA private key
func generateKey() ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	if cerr := priv.Validate(); cerr != nil {
		return nil, cerr
	}

	pemdata := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	return pemdata, nil
}

func generateCert(pempriv []byte) ([]byte, error) {

	snLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	sn, err := rand.Int(rand.Reader, snLimit)

	if err != nil {
		log.Debug("Could not generate certificate serial number")
	}

	ca := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Country:            []string{""},
			Organization:       []string{""},
			OrganizationalUnit: []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		SubjectKeyId:          []byte{},
		BasicConstraintsValid: true,
		//IsCA:        false,
		//ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		//KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	block, _ := pem.Decode(pempriv)

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Errorf("Could not parse private key: %s", err.Error())
		return nil, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, ca, ca, priv.Public(), priv)
	if err != nil {
		return nil, err
	}

	certpem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	return certpem, nil
}
