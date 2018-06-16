package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ed25519"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginFlow_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready := make(chan struct{})
	go run(ctx, ready)
	<-ready
	time.Sleep(time.Second) // give some time to the server to start

	var b []byte
	token := uuid.New().String()

	// client: request qrcode
	{
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/1.0/qrCode?token=%s", token), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var qrCodeResp qrCodeResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&qrCodeResp))

		b, err = base64.StdEncoding.DecodeString(qrCodeResp.QRCode)
		require.NoError(t, err)
	}

	errCh := make(chan error, 0)
	// client: wait to get authenticated
	wg := new(sync.WaitGroup)
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(errCh)

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/1.0/isAuthenticated?token=%s", token), nil)
			if err != nil {
				errCh <- err
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errCh <- err
				return
			}

			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("Response status: %d", resp.StatusCode)
				return
			}

			var i interface{}
			err = json.NewDecoder(resp.Body).Decode(&i)
			for err == nil {
				log.Println(i)
				err = json.NewDecoder(resp.Body).Decode(&i)
			}
			if err != nil && err != io.EOF {
				errCh <- fmt.Errorf("Reading response body: %v", err)
			}
		}()
	}

	// device: decode qrcode
	{
		body, contentType := buildForm(t, b)
		req, err := http.NewRequest("POST", "http://api.qrserver.com/v1/read-qr-code/", body)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var qrResp []qrAPIResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&qrResp))

		require.Len(t, qrResp, 1)
		require.Len(t, qrResp[0].Symbol, 1)

		assert.Equal(t, token, qrResp[0].Symbol[0].Data)
		assert.Nil(t, qrResp[0].Symbol[0].Error)
	}

	// device: sign token and authenticate
	{
		signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(token)))
		authReq := authRequest{
			DeviceID:  testDevice.id,
			Message:   token,
			Signature: signature,
		}
		payload, err := json.Marshal(authReq)
		require.NoError(t, err)

		baseURL := "http://localhost:8080/api/1.0/authenticate"
		req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(payload))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	for err := range errCh {
		t.Fatal(err)
	}
	wg.Wait()
}

func Test_LearningTest_QRServerAPI(t *testing.T) {
	var b []byte
	token := uuid.New().String()

	//request qrcode
	{
		resp, err := http.Get(fmt.Sprintf("http://api.qrserver.com/v1/create-qr-code/?data=%s&size=100x100", token))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

		b, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
	}

	// decode qrcode
	{
		body, contentType := buildForm(t, b)
		req, err := http.NewRequest("POST", "http://api.qrserver.com/v1/read-qr-code/", body)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		var qrResp []qrAPIResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&qrResp))

		log.Printf("response = %+v\n", qrResp)
		require.Len(t, qrResp, 1)
		require.Len(t, qrResp[0].Symbol, 1)
		symbol := qrResp[0].Symbol[0]

		assert.Equal(t, token, symbol.Data)
		require.Nil(t, symbol.Error)
	}
}

func buildForm(t *testing.T, b []byte) (io.Reader, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "qrcode.png")
	require.NoError(t, err)
	_, err = io.Copy(part, bytes.NewReader(b))
	require.NoError(t, err)

	part, err = writer.CreateFormField("MAX_FILE_SIZE")
	require.NoError(t, err)
	_, err = io.Copy(part, strings.NewReader("1048576"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

type qrAPIResponse struct {
	Type   string
	Symbol []symbol
}

type symbol struct {
	Seq   int
	Data  string
	Error interface{}
}
