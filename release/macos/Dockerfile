# initialize from the image

FROM fedora:32

# update package repositories

RUN dnf update -y

# install tools

RUN dnf install -y gcc-c++ git make cpio findutils openssl xar

# build bomutils from source

RUN git clone https://github.com/hogliux/bomutils.git
RUN cd bomutils && make && make install
