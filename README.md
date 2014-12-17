goassest
========
go assets tool

install
```
go get -u github.com/hidu/goassest
```

useage
```
goassest [go assest dir]
```
output file is `assest.go` in assest dir  

assest conf is: `assest.json`:
```
{
  "assestDir":"res",
  "destName":"serve/assest.go",
  "packageName":"serve"
}
```