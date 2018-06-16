# Fun

```console
$ go run main.go

$ curl localhost:8080/api/1.0/qrCode?token=funfunfun | jq -r .qrCode | base64 -D > qrCode.png
```

## endpoints

* GET `/api/1.0/qrCode?token={token}` - returns a response like `{"qrCode":"iVBORw0KG..."}, where qrCode is a base64 encoded png

* GET `/api/1.0/isAuthenticated?token={token}` - it's a long-polled request, returns a response like `{"login":"success"}` when available.

* POST `/api/1.0/authenticate?deviceId={deviceId}&message={message}&signature={signature}` - message is the qr encoded data returned by `/api/1.0/qrCode` endpoint. Signature is the signed message by the device key

## implementation

For now, the token is the qrCode encoded data.

