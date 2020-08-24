# initialize from the image

FROM fedora:25

# update package repositories

RUN dnf update -y

# install tools

RUN dnf install -y gcc gcc-c++ git make gzip cpio findutils autoconf libxml2-devel openssl-devel

# build bomutils from source

RUN git clone https://github.com/hogliux/bomutils.git
RUN cd bomutils && make && make install

# build xar from source
# (xar in Fedora is old and its results don't work in macOS)

RUN git clone https://github.com/mackyle/xar.git
RUN cd xar && cd xar && ./autogen.sh --noconfigure && ./configure && make && make install
