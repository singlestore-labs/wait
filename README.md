# wait - wait for a condition to become true

[![GoDoc](https://godoc.org/github.com/singlestore-labs/wait?status.svg)](https://pkg.go.dev/github.com/singlestore-labs/wait)
![unit tests](https://github.com/singlestore-labs/wait/actions/workflows/go.yml/badge.svg)
[![report card](https://goreportcard.com/badge/github.com/singlestore-labs/wait)](https://goreportcard.com/report/github.com/singlestore-labs/wait)
[![codecov](https://codecov.io/gh/singlestore-labs/wait/branch/main/graph/badge.svg)](https://codecov.io/gh/singlestore-labs/wait)

Install:

	go get github.com/singlestore-labs/wait

---

Wait provides one function: `For`.

```go
	err := wait.For(func() (bool, error) {
		err := trySomething()
		if err != nil {
			if isFatal(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
```

It calls its function argument over and over until:

- a timeout occurs (failure)
- the bool becomes true (success or failure depending on error)
- if ExitOnError(true) and the error becomes non nil (failure) 


