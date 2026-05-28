package rest

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/spacemonkeygo/httpsig"
)

func signRequest(req *http.Request, body []byte, apiKey, secret string) error {
	if apiKey == "" {
		return nil
	}
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	} else {
		body = []byte{}
	}

	// Hub REST uses X-API-KEY; also sign for Rafay gateway compatibility (paasctl/rctl).
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("X-RAFAY-API-KEYID", apiKey)
	req.Header.Set("date", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("content-md5", bodyChecksum(body))
	req.Header.Set("nonce", strconv.Itoa(rand.Int()))

	if secret == "" {
		secret = apiKey
	}
	signer := httpsig.NewHMACSHA256Signer(apiKey, []byte(secret),
		[]string{"content-md5", "date", "host", "nonce"})
	return signer.Sign(req)
}

func bodyChecksum(body []byte) string {
	hash := md5.New()
	hash.Write(body)
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
