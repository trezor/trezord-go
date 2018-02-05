# initialize from the image

FROM fedora:24

# update package repositories

RUN dnf update -y

# install tools

RUN dnf install -y cmake make wget
RUN dnf install -y osslsigncode mingw32-nsis
