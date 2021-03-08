#!/bin/sh
unset -v progdir
case "${0}" in
*/*) progdir="${0%/*}";;
*) progdir=.;;
esac
"${progdir}/list_nordicenergy_go_files.sh" | xargs goimports "$@"
