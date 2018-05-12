# Fun

```console
$ go run main.go
```

```console
$ curl localhost:8080/api/1.0/qrCode?token= | jq -r .qrCode | base64 -D > qrCode.png
```

## endpoints

* GET `/api/1.0/qrCode?token={token}` - returns a response like `{"qrCode":"iVBORw0KG..."}, where qrCode is a base64 encoded png

* GET `/api/1.0/loggedIn?token={token}` - it's a long-polled request, returns a response like `{"login":"success"}` when available.

* POST `/api/1.0/login?accID={accID}&secureToken={secureToken}` - secureToken is the encrypted qrCode data encrypted by the device priv key
