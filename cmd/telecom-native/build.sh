go build -o telecom.a -buildmode=c-archive main.go
mv telecom.a libtelecom.a
