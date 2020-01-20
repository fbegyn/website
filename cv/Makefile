all: render

render:
	pandoc cv.html -c cv.css -t html5 -o cv.pdf \
		-V margin-top=3 -V margin-left=3 -V margin-right=3 -V margin-bottom=3 \
		-V lettersize=a4
