from strangesast/fwlib as base

from golang as builder
workdir /usr/src/app
copy . .
run go build -o fwlib-go .

from base
copy --from=builder /usr/src/app/fwlib-go /fwlib-go
cmd ["/fwlib-go"]
