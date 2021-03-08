{{VER=2.0.0}}
{{REL=0}}
# SPEC file overview:
# https://docs.fedoraproject.org/en-US/quick-docs/creating-rpm-packages/#con_rpm-spec-file-overview
# Fedora packaging guidelines:
# https://docs.fedoraproject.org/en-US/packaging-guidelines/

Name:		nordicenergy
Version:	{{ VER }}
Release:	{{ REL }}
Summary:	nordicenergy blockchain validator node program

License:	MIT
URL:		https://nordicenergy.net
Source0:	%{name}-%{version}.tar
BuildArch: x86_64
Packager: Leo Chen <leo@hamrony.net>
Requires(pre): shadow-utils
Requires: systemd-rpm-macros jq

%description
nordicenergy is a sharded, fast finality, low fee, PoS public blockchain.
This package contains the validator node program for nordicenergy blockchain.

%global debug_package %{nil}

%prep
%setup -q

%build
exit 0

%check
./nordicenergy --version
exit 0

%pre
getent group nordicenergy >/dev/null || groupadd -r nordicenergy
getent passwd nordicenergy >/dev/null || \
   useradd -r -g nordicenergy -d /home/nordicenergy -m -s /sbin/nologin \
   -c "nordicenergy validator node account" nordicenergy
mkdir -p /home/nordicenergy/.ngy/blskeys
mkdir -p /home/nordicenergy/.config/rclnet
chown -R nordicenergy.nordicenergy /home/nordicenergy
exit 0


%install
install -m 0755 -d ${RPM_BUILD_ROOT}/usr/sbin ${RPM_BUILD_ROOT}/etc/systemd/system ${RPM_BUILD_ROOT}/etc/sysctl.d ${RPM_BUILD_ROOT}/etc/nordicenergy
install -m 0755 -d ${RPM_BUILD_ROOT}/home/nordicenergy/.config/rclnet
install -m 0755 nordicenergy ${RPM_BUILD_ROOT}/usr/sbin/
install -m 0755 nordicenergy-setup.sh ${RPM_BUILD_ROOT}/usr/sbin/
install -m 0755 nordicenergy-rclnet.sh ${RPM_BUILD_ROOT}/usr/sbin/
install -m 0644 nordicenergy.service ${RPM_BUILD_ROOT}/etc/systemd/system/
install -m 0644 nordicenergy-sysctl.conf ${RPM_BUILD_ROOT}/etc/sysctl.d/99-nordicenergy.conf
install -m 0644 rclnet.conf ${RPM_BUILD_ROOT}/etc/nordicenergy/
install -m 0644 nordicenergy.conf ${RPM_BUILD_ROOT}/etc/nordicenergy/
exit 0

%post
%systemd_user_post %{name}.service
%sysctl_apply %{name}-sysctl.conf
exit 0

%preun
%systemd_user_preun %{name}.service
exit 0

%postun
%systemd_postun_with_restart %{name}.service
exit 0

%files
/usr/sbin/nordicenergy
/usr/sbin/nordicenergy-setup.sh
/usr/sbin/nordicenergy-rclnet.sh
/etc/sysctl.d/99-nordicenergy.conf
/etc/systemd/system/nordicenergy.service
/etc/nordicenergy/nordicenergy.conf
/etc/nordicenergy/rclnet.conf
/home/nordicenergy/.config/rclnet

%config(noreplace) /etc/nordicenergy/nordicenergy.conf
%config /etc/nordicenergy/rclnet.conf
%config /etc/sysctl.d/99-nordicenergy.conf 
%config /etc/systemd/system/nordicenergy.service

%doc
%license



%changelog
* Wed Aug 26 2020 Leo Chen <leo at nordicenergy dot net> 2.3.5
   - get version from git tag
   - add %config macro to keep edited config files

* Tue Aug 18 2020 Leo Chen <leo at nordicenergy dot net> 2.3.4
   - init version of the nordicenergy node program

