.PHONY: run
run:
	depviz render -f | dot -Tsvg > depviz.svg
	depviz render -t orphans -f | dot -Tsvg > orphans.svg
	cat orphans.svg | grep -s text || rm orphans.svg

.PHONY: preview
preview:
	depviz render -f | dot -Tpng | imgcat
