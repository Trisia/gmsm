package websvrcase

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/tjfoc/gmsm/x509"

	"github.com/tjfoc/gmsm/gmtls"
)

const (
	rsaCertPath = "certs/rsa_sign.cer"
	rsaKeyPath  = "certs/rsa_sign_key.pem"

	sm2SignCertPath = "certs/sm2_sign_cert.cer"
	sm2SignKeyPath  = "certs/sm2_sign_key.pem"
	sm2EncCertPath  = "certs/sm2_enc_cert.cer"
	sm2EncKeyPath   = "certs/sm2_enc_key.pem"
	SM2CaCertPath   = "certs/SM2_CA.cer"
)

// RSA配置
func loadRsaConfig() (*gmtls.Config, error) {
	cert, err := gmtls.LoadX509KeyPair(rsaCertPath, rsaKeyPath)
	if err != nil {
		return nil, err
	}
	return &gmtls.Config{Certificates: []gmtls.Certificate{cert}}, nil
}

// SM2配置
func loadSM2Config() (*gmtls.Config, error) {
	sigCert, err := gmtls.LoadX509KeyPair(sm2SignCertPath, sm2SignKeyPath)
	if err != nil {
		return nil, err
	}
	encCert, err := gmtls.LoadX509KeyPair(sm2EncCertPath, sm2EncKeyPath)
	if err != nil {
		return nil, err
	}
	return &gmtls.Config{
		GMSupport:    &gmtls.GMSupport{},
		Certificates: []gmtls.Certificate{sigCert, encCert},
	}, nil
}

// 切换GMSSL/TSL
func loadAutoSwitchConfig() (*gmtls.Config, error) {
	rsaKeypair, err := gmtls.LoadX509KeyPair(rsaCertPath, rsaKeyPath)
	if err != nil {
		return nil, err
	}
	sigCert, err := gmtls.LoadX509KeyPair(sm2SignCertPath, sm2SignKeyPath)
	if err != nil {
		return nil, err
	}
	encCert, err := gmtls.LoadX509KeyPair(sm2EncCertPath, sm2EncKeyPath)
	if err != nil {
		return nil, err

	}
	return gmtls.NewBasicAutoSwitchConfig(&sigCert, &encCert, &rsaKeypair)
}
func ServerRun() {
	//config, err := loadRsaConfig()
	//config, err := loadSM2Config()
	config, err := loadAutoSwitchConfig()
	if err != nil {
		panic(err)
	}

	ln, err := gmtls.Listen("tcp", ":443", config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "hello\n")
	})
	fmt.Println(">> HTTP Over [GMSSL/TLS] running...")
	err = http.Serve(ln, nil)
	if err != nil {
		panic(err)
	}
}
func ClientRun() {
	var config = tls.Config{
		MaxVersion:         gmtls.VersionTLS12,
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", "localhost:443", &config)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	req := []byte("GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Connection: close\r\n\r\n")
	conn.Write(req)

	buff := make([]byte, 1024)
	for {
		n, _ := conn.Read(buff)
		if n <= 0 {
			break
		} else {
			fmt.Printf("%s", buff[0:n])
		}
	}
	fmt.Println()
	end <- true
}
func gmClientRun() {

	// 信任的根证书
	certPool := x509.NewCertPool()
	cacert, err := ioutil.ReadFile(SM2CaCertPath)
	if err != nil {
		log.Fatal(err)
	}
	certPool.AppendCertsFromPEM(cacert)

	config := &gmtls.Config{
		GMSupport: &gmtls.GMSupport{},
		RootCAs:   certPool,
	}

	conn, err := gmtls.Dial("tcp", "localhost:443", config)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	req := []byte("GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Connection: close\r\n\r\n")
	_, _ = conn.Write(req)
	buff := make([]byte, 1024)
	for {
		n, _ := conn.Read(buff)
		if n <= 0 {
			break
		} else {
			fmt.Printf("%s", buff[0:n])
		}
	}
	fmt.Println()
	end <- true
}

var end chan bool

func Test(t *testing.T) {
	end = make(chan bool, 64)
	go ServerRun()
	time.Sleep(1000000)
	go ClientRun()
	<-end
	go gmClientRun()
	<-end
}
