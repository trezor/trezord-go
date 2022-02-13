# initialize from the image

FROM fedora:32

# update package repositories

RUN dnf update -y

# install dependencies for Linux packaging

RUN dnf install -y ruby-devel rubygems rpm-build
RUN dnf install -y autoconf automake libtool make
RUN dnf install -y git
RUN gem install fpm --no-document

# install package signing tools
RUN dnf install -y rpm-sign
RUN ln -s gpg2 /usr/bin/gpg
