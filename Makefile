fuzz: clean
	go-fuzz-build github.com/homelight/worksheets/fuzz
	go-fuzz -bin=./worksheets-fuzz.zip -workdir=fuzz/

clean:
	rm -f worksheets-fuzz.zip
	find fuzz/corpus/ -type f -not -name 'corpus_[0-9]*' -exec rm {} \;
	rm -Rf fuzz/crashers/
	rm -Rf fuzz/suppressions/
