TOP:=$(realpath ..)
export CGO_CFLAGS:=-I$(TOP)/bls/include -I$(TOP)/mcl/include -I/usr/local/opt/openssl/include
export CGO_LDFLAGS:=-L$(TOP)/bls/lib -L/usr/local/opt/openssl/lib
export LD_LIBRARY_PATH:=$(TOP)/bls/lib:$(TOP)/mcl/lib:/usr/local/opt/openssl/lib
export LIBRARY_PATH:=$(LD_LIBRARY_PATH)
export DYLD_FALLBACK_LIBRARY_PATH:=$(LD_LIBRARY_PATH)
export GO111MODULE:=on
PKGNAME=nordicenergy
VERSION?=$(shell git tag -l --sort=-v:refname | head -n 1 | tr -d v)
RELEASE?=$(shell git describe --long | cut -f2 -d-)
RPMBUILD=$(HOME)/rpmbuild
DEBBUILD=$(HOME)/debbuild
SHELL := bash

.PHONY: all help libs exe race trace-pointer debug debug-kill test test-go test-api test-api-attach linux_static deb_init deb_build deb debpub_dev debpub_prod rpm_init rpm_build rpm rpmpub_dev rpmpub_prod clean distclean

all: libs
	bash ./scripts/go_executable_build.sh -S

help:
	@echo "all - build the nordicenergy binary & bootnode along with the MCL & BLS libs (if necessary)"
	@echo "libs - build only the MCL & BLS libs (if necessary) "
	@echo "exe - build the nordicenergy binary & bootnode"
	@echo "race - build the nordicenergy binary & bootnode with race condition checks"
	@echo "trace-pointer - build the Nordic Energy binary & bootnode with pointer analysis"
	@echo "debug - start a localnet with 2 shards (s0 rpc endpoint = localhost:9599; s1 rpc endpoint = localhost:9598)"
	@echo "debug-kill - force kill the localnet"
	@echo "clean - remove node files & logs created by localnet"
	@echo "distclean - remove node files & logs created by localnet, and all libs"
	@echo "test - run the entire test suite (go test & Node API test)"
	@echo "test-go - run the go test (with go lint, fmt, imports, mod, and generate checks)"
	@echo "test-rpc - run the rpc tests"
	@echo "test-rpc-attach - attach onto the rpc testing docker container for inspection"
	@echo "test-rosetta - run the rosetta tests"
	@echo "test-rosetta-attach - attach onto the rosetta testing docker container for inspection"
	@echo "linux_static - static build the nordicenergy binary & bootnode along with the MCL & BLS libs (for linux)"
	@echo "rpm - build a Nordic Energy RPM pacakge"
	@echo "rpmpub_dev - publish Nordic Energy RPM package to development repo"
	@echo "rpmpub_prod - publish Nordic Energy RPM package to production repo"
	@echo "deb - build a Nordic Energy Debian package"
	@echo "debpub_dev - publish Nordic Energy Debian package to development repo"
	@echo "debpub_prod - publish Nordic Energy Debian package to production repo"

libs:
	make -C $(TOP)/mcl -j8
	make -C $(TOP)/bls BLS_SWAP_G=1 -j8

exe:
	bash ./scripts/go_executable_build.sh -S

race:
	bash ./scripts/go_executable_build.sh -r

trace-pointer:
	bash ./scripts/go_executable_build.sh -t

debug:
	bash ./test/debug.sh

debug-kill:
	bash ./test/kill_node.sh

clean:
	rm -rf ./tmp_log*
	rm -rf ./.dht*
	rm -rf ./db-*
	rm -rf ./latest
	rm -f ./*.rlp
	rm -rf ~/rpmbuild
	rm -f coverage.txt

distclean: clean
	make -C $(TOP)/mcl clean
	make -C $(TOP)/bls clean

go-get:
	source ./scripts/setup_bls_build_flags.sh
	go get -v ./...

test:
	bash ./test/all.sh

test-go:
	bash ./test/go.sh

test-rpc:
	bash ./test/rpc.sh run

test-rpc-attach:
	bash ./test/rpc.sh attach

test-rosetta:
	bash ./test/rosetta.sh run

test-rosetta-attach:
	bash ./test/rosetta.sh attach

linux_static:
	make -C $(TOP)/mcl -j8
	make -C $(TOP)/bls minimised_static BLS_SWAP_G=1 -j8
	bash ./scripts/go_executable_build.sh -s

deb_init:
	rm -rf $(DEBBUILD)
	mkdir -p $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/{etc/systemd/system,usr/sbin,etc/sysctl.d,etc/nordicenergy}
	cp -f bin/nordicenergy $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/usr/sbin/
	bin/nordicenergy dumpconfig $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/etc/nordicenergy/nordicenergy.conf
	cp -f scripts/package/rclnet.conf $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/etc/nordicenergy/
	cp -f scripts/package/nordicenergy.service $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/etc/systemd/system/
	cp -f scripts/package/nordicenergy-setup.sh $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/usr/sbin/
	cp -f scripts/package/nordicenergy-rclnet.sh $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/usr/sbin/
	cp -f scripts/package/nordicenergy-sysctl.conf $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/etc/sysctl.d/99-nordicenergy.conf
	cp -r scripts/package/deb/DEBIAN $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)
	VER=$(VERSION)-$(RELEASE) scripts/package/templater.sh scripts/package/deb/DEBIAN/control > $(DEBBUILD)/$(PKGNAME)-$(VERSION)-$(RELEASE)/DEBIAN/control

deb_build:
	(cd $(DEBBUILD); dpkg-deb --build $(PKGNAME)-$(VERSION)-$(RELEASE)/)

deb: deb_init deb_build

debpub_dev: deb
	cp scripts/package/deb/dev.aptly.conf ~/.aptly.conf
	./scripts/package/publish-repo.sh -p dev -n deb -s $(DEBBUILD)

debpub_prod: deb
	cp scripts/package/deb/prod.aptly.conf ~/.aptly.conf
	./scripts/package/publish-repo.sh -p prod -n deb -s $(DEBBUILD)

rpm_init:
	rm -rf $(RPMBUILD)
	mkdir -p $(RPMBUILD)/{SOURCES,SPECS,BUILD,RPMS,BUILDROOT,SRPMS}
	mkdir -p $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	cp -f bin/nordicenergy $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	bin/nordicenergy dumpconfig $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)/nordicenergy.conf
	cp -f scripts/package/nordicenergy.service $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	cp -f scripts/package/nordicenergy-setup.sh $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	cp -f scripts/package/nordicenergy-rclnet.sh $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	cp -f scripts/package/rclnet.conf $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	cp -f scripts/package/nordicenergy-sysctl.conf $(RPMBUILD)/SOURCES/$(PKGNAME)-$(VERSION)
	VER=$(VERSION) REL=$(RELEASE) scripts/package/templater.sh scripts/package/rpm/nordicenergy.spec > $(RPMBUILD)/SPECS/nordicenergy.spec
	(cd $(RPMBUILD)/SOURCES; tar cvf $(PKGNAME)-$(VERSION).tar $(PKGNAME)-$(VERSION))

rpm_build:
	rpmbuild --target x86_64 -bb $(RPMBUILD)/SPECS/nordicenergy.spec

rpm: rpm_init rpm_build
	rpm --addsign $(RPMBUILD)/RPMS/x86_64/$(PKGNAME)-$(VERSION)-$(RELEASE).x86_64.rpm

rpmpub_dev: rpm
	./scripts/package/publish-repo.sh -p dev -n rpm -s $(RPMBUILD)

rpmpub_prod: rpm
	./scripts/package/publish-repo.sh -p prod -n rpm -s $(RPMBUILD)