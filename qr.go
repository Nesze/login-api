package main

import (
	"bytes"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

type qrService struct {
}

func (q qrService) generatePNG(token string) ([]byte, error) {
	qrCode, err := qr.Encode(token, qr.M, qr.Auto)
	if err != nil {
		return nil, err
	}

	qrCode, err = barcode.Scale(qrCode, 100, 100)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := png.Encode(&b, qrCode); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
