go-get-package:
	go get -u github.com/magic-lib/go-plat-utils@master
	go get -u github.com/magic-lib/go-plat-cache@master
	go get -u github.com/magic-lib/go-plat-mysql@master
	go get -u github.com/magic-lib/go-plat-retry@master
	go get -u github.com/magic-lib/go-servicekit/tracer@master
	go mod tidy