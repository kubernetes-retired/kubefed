# spec file for building the kubefed-client rpm 

#debuginfo not supported with Go
%global debug_package %{nil}

# modifying the Go binaries breaks the DWARF debugging
%global __os_install_post %{_rpmconfigdir}/brp-compress

# %commit and %os_git_vars are intended to be set by tito custom builders provided
# in the .tito/lib directory. The values in this spec file will not be kept up to date.
%{!?commit: %global commit HEAD }
%global shortcommit %(c=%{commit}; echo ${c:0:7})
# os_git_vars needed to run hack scripts during rpm builds
%{!?os_git_vars: %global os_git_vars OS_GIT_VERSION='' OS_GIT_COMMIT='' OS_GIT_MAJOR='' OS_GIT_MINOR='' OS_GIT_TREE_STATE='' }

%if 0%{?skip_build}
%global do_build 0
%else
%global do_build 1
%endif
%if 0%{?skip_prep}
%global do_prep 0
%else
%global do_prep 1
%endif

%if 0%{?fedora} || 0%{?epel}
%global need_redistributable_set 0
%else
# Due to library availability, redistributable builds only work on x86_64
%ifarch x86_64
%global need_redistributable_set 1
%else
%global need_redistributable_set 0
%endif
%endif
%{!?make_redistributable: %global make_redistributable %{need_redistributable_set}}

#
# Customize from here.
#

%global golang_version 1.12
%{!?version: %global version 0.1.0}
%{!?release: %global release 1}

%global package_name kubefed-client
%global product_name Kubefed Client
%global import_path github.com/openshift/kubefed # GO_PACKAGE

Name:           %{package_name}
Version:        %{version}
Release:        %{release}%{?dist}
Summary:        Kubefed Client (kubefedctl)
License:        ASL 2.0
URL:            https://github.com/openshift/kubefed

Source0:        https://%{import_path}/archive/%{commit}/kubefed-%{version}.tar.gz
BuildRequires:  golang >= %{golang_version}
Provides:       kubefedctl
# If go_arches not defined fall through to implicit golang archs
%if 0%{?go_arches:1}
ExclusiveArch:  %{go_arches}
%else
ExclusiveArch:  x86_64 aarch64 ppc64le s390x
%endif

### AUTO-BUNDLED-GEN-ENTRY-POINT

%description
This package provides the kubefed-client binary (kubefedctl) to interact with the kubefed controller

%prep
GOPATH=$RPM_BUILD_DIR/go
rm -rf $GOPATH
mkdir -p $GOPATH/src/sigs.k8s.io/kubefed
cd $RPM_BUILD_DIR
rm -rf kubefed-client*
tar -xzmf %{_sourcedir}/kubefed-client*
cd kubefed-client*
DIR=$RPM_BUILD_DIR/kubefed-client*
mv $DIR/*  $GOPATH/src/sigs.k8s.io/kubefed/
ln -s $GOPATH/src/sigs.k8s.io/kubefed/kubefed $DIR

%build
export GOPATH=$RPM_BUILD_DIR/go
cd $GOPATH/src/sigs.k8s.io/kubefed
%if 0%{do_build}
%if 0%{make_redistributable}
# Create Binaries for all internally defined arches
%{os_git_vars} make -f openshift/Makefile.ci kubefedctl
%else
# Create Binaries only for building arch
%ifarch x86_64
BUILD_PLATFORM="linux/amd64"
%endif
%ifarch %{ix86}
BUILD_PLATFORM="linux/386"
%endif
%ifarch ppc64le
BUILD_PLATFORM="linux/ppc64le"
%endif
%ifarch %{arm} aarch64
BUILD_PLATFORM="linux/arm64"
%endif
%ifarch s390x
BUILD_PLATFORM="linux/s390x"
%endif
OS_ONLY_BUILD_PLATFORMS="${BUILD_PLATFORM}" %{os_git_vars} make -f openshift/Makefile.ci kubefedctl
%endif
%endif

%install
install -d %{buildroot}%{_bindir}

install -p -m 755 $RPM_BUILD_DIR/go/src/sigs.k8s.io/kubefed/bin/kubefedctl %{buildroot}%{_bindir}/kubefedctl

# EXAMPLE: Install tests
# install -d %{buildroot}%{_libexecdir}/%{name}
# install -p -m 755 _output/local/bin/${PLATFORM}/extended.test %{buildroot}%{_libexecdir}/%{name}/

# EXAMPLE: Install other files
# install -p -m 0755 _output/local/bin/${PLATFORM}/sdn-cni-plugin %{buildroot}/opt/cni/bin/openshift-sdn

%files
%doc $RPM_BUILD_DIR/go/src/sigs.k8s.io/kubefed/README.md
%license $RPM_BUILD_DIR/go/src/sigs.k8s.io/kubefed/LICENSE
%{_bindir}/kubefedctl
# EXAMPLE: Managing configuration
# %defattr(-,root,root,0700)
# %dir %config(noreplace) %{_sysconfdir}/origin
# %ghost %dir %config(noreplace) %{_sysconfdir}/origin
# %ghost %config(noreplace) %{_sysconfdir}/origin/.config_managed

%pre

%changelog
* Tue Jun 25 2019 Aniket Bhat <anbhat@redhat.com>
- Initial spec file for building kubefed-client rpm.
- Change the name of the make target from build-cross to kubefedctl
