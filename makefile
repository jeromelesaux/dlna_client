CC=go
RM=rm
MV=mv


SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
GOOS=linux
GOARCH=amd64

VERSION:=1.0
PREVIOUS_VERSION=$(shell echo $$((${VERSION} - 1)))

APP=DlnaClient

BUILD_TIME=`date +%FT%T%z`
PACKAGES := github.com/huin/goupnp

LIBS=

LDFLAGS=-ldflags "-w -race"



test: $(APP)
		@GOOS=${GOOS} GOARCH=${GOARCH} go test ./...
		@echo " Tests OK."

$(APP): organize $(SOURCES)
		@echo "    Compilation des sources ${BUILD_TIME}"
		@GOOS=${GOOS} GOARCH=${GOARCH} go build ${LDFLAGS} -o ${APP}-${VERSION} $(SOURCEDIR)/main.go
		@echo "    ${APP}-${VERSION} generated."

organize: audit
		@echo "    Go FMT"
		@$(foreach element,$(SOURCES),go fmt $(element);)

audit: deps
		@go tool vet -all . 2> audit.log &
		@echo "    Audit effectue"

deps: init
		@echo "    Download packages"
		@$(foreach element,$(PACKAGES),go get -d -v -insecure $(element);)
		@go install

init: clean
		@echo "    Init of the project"
		@echo "    Version :: ${VERSION}"

execute:
		./${APP}-${VERSION}  -pattern "star wars"

clean:
		@if [ -f "${APP}-${VERSION}" ] ; then rm ${APP}-${VERSION} ; fi
		@echo "    Nettoyage effectuee"


package-zip:  ${APP}
		@zip -r ${APP}-${GOOS}-${GOARCH}-${VERSION}.zip ./${APP}-${VERSION}
		@echo "    Archive ${APP}-${GOOS}-${GOARCH}-${VERSION}.zip created"

