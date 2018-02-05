# initialize from the image
FROM debian:stretch

# update package repositories
RUN apt-get update

# install tools

RUN apt-get install -y autoconf build-essential libxml2-dev libssl1.0-dev zlib1g-dev git cpio

# build additional tools

RUN git clone https://github.com/hogliux/bomutils.git
RUN cd bomutils && make -j4 && make install
RUN git clone https://github.com/mackyle/xar.git
RUN cd xar && cd xar && ./autogen.sh --noconfigure && ./configure && make -j4 && make install
