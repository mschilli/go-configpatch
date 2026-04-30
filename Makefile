all:
	cd cli; go build && cp cli ~/bin/configpatch

mkrefs:
	cd test_data/cases; for i in *; do cd $$i; ../../bin/mkref; cd ..; done

fmt:
	gofmt -w *.go 
