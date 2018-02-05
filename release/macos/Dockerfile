# initialize from the image

FROM fedora:25

# update package repositories

RUN dnf update -y

# install tools

RUN dnf install -y gcc gcc-c++ git make gzip cpio findutils xar

# build bomutils from source

RUN git clone https://github.com/hogliux/bomutils.git
RUN cd bomutils && make && make install
