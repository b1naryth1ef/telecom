go build -o telecom.so -buildmode=c-shared github.com/b1naryth1ef/telecom/cmd/telecom-native
go build -o telecom.a -buildmode=c-archive github.com/b1naryth1ef/telecom/cmd/telecom-native

sudo mv telecom.so /usr/local/lib/libtelecom.so
sudo mv telecom.a /usr/local/lib/libtelecom.a
sudo mv telecom.h /usr/local/include/telecom.h
