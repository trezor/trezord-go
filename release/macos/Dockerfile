# initialize from the image

FROM fedora:39

# update package repositories

RUN dnf update -y

# install tools

RUN dnf install -y gcc gcc-c++ git make gzip cpio findutils autoconf libxml2-devel openssl-devel cargo gpg

# build bomutils and xar from sources with fixes

RUN git clone https://github.com/tpoechtrager/xar
RUN cd xar && cd xar && ./autogen.sh --noconfigure && ./configure && make && make install

RUN git clone https://github.com/BC-SECURITY/bomutils
RUN cd bomutils && CFLAGS="-fsigned-char" make && make install

RUN cargo install apple-codesign
