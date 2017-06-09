# citemicroservices

Implementation of the CITE architexture microservices specified [here](https://github.com/cite-architecture/citemicroservices).


## Current version: 1.0.4

Versions of binary releases follow [semantic versioning](http://semver.org/) guidelines.

See [release notes](releases.md)




## Get Started
1. Clone the repository or download and unpack the zipped repository. (You can rename it afterwards if you like.)
2. Open a Terminal/Commandline and `cd` into the new folder.
3. On Mac/Linux: start it up with `./citeMicros-VERSION` / on Windows: doubleclick on `WinCiteMircros-VERSION.exe`

## Trouble-shooting

You might have to tell your Operating system that `./citeMicros-VERSION` is an executable with `chmod +x citeMicros-VERSION`

## Test it with your favourite browser

1. http://localhost:8080/cite
2. http://localhost:8080/texts
3. http://localhost:8080/texts/
4. http://localhost:8080/texts/urn:cts:citeArch:groupA.work1.ed1:1-2
5. http://localhost:8080/texts/urns/urn:cts:citeArch:groupA.work1.ed1:1-2
6. http://localhost:8080/texts/first/urn:cts:citeArch:groupA.work1.ed1:1-2
7. http://localhost:8080/texts/last/urn:cts:citeArch:groupA.work1.ed1:1-2
8. http://localhost:8080/texts/next/urn:cts:citeArch:groupA.work1.ed1:3.2
9. http://localhost:8080/texts/previous/urn:cts:citeArch:groupA.work1.ed1:3.2

## Test it with your own CEX

1. Change the "cex_source" in `config.json` or try it with my CEX file
2. Execute the http-request like above but add `[the_name_of_your_cex]` in front of it
3. For instance, http://localhost:8080/million/texts/
4. If you name your cex files `texts.cex` won't work with this implementation of the microservices.

## Modify it to meet your needs:

`config.json` is pretty much self-explicable.
