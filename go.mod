module stock

go 1.15

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/ipfs/go-log/v2 v2.1.2
	github.com/liyue201/golib v0.0.0-20210225015707-e527cc337867
)

replace (
	github.com/ipfs/go-log/v2 v2.1.2 => github.com/liyue201/go-log/v2 v2.1.2-0.20210312022747-08ac38b3a792
)