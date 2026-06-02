# Commands

## Run

**Frontend:**

```bash
cd ./frontend/
npm start
```

**Backend Linux:**

```bash
cd ./backend/
make run
```

**Backend Windows:**

```bat
cd .\backend\
go run .\cmd\server
```

## Test

**Frontend:**

```bash
cd ./frontend/
npm run test:ci
```

**Backend Linux:**

```bash
cd ./backend/
make test
```

**Backend Windows:**

```bat
cd ./backend/
go test .\...
```

## Publish

**Frontend:**

```bash
cd ./frontend/
rm -r ./dist/frontend/browser
npm run build
scp -r ./dist/frontend/browser/* ubuntu@${SERVER}:/opt/fast-bypass/www/
```

**Backend Linux:**

```bash
cd ./backend/
rm server
# params
go build -o server ./cmd/server
scp ./server ubuntu@${SERVER}:/opt/fast-bypass/bin/
```

**Backend Windows:**

```bat
cd .\backend\
rm server
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o server .\cmd\server
scp .\server ubuntu@%SERVER%:/opt/fast-bypass/bin/
```

**On Server**
```bash
chmod +x /opt/fast-bypass/bin/server
```

