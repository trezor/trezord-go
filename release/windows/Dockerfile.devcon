FROM debian:stretch

RUN apt-get update && \
  apt-get install -y \
    g++-mingw-w64

ADD devcon /devcon

WORKDIR /devcon
RUN x86_64-w64-mingw32-windmc msg.mc

WORKDIR /devcon/build-x86_64
RUN x86_64-w64-mingw32-windres ../devcon.rc rc.so
RUN x86_64-w64-mingw32-g++ -municode -Wno-write-strings \
    -DWIN32_LEAN_AND_MEAN=1 -DUNICODE -D_UNICODE \
    ../*.cpp rc.so \
    -lsetupapi -lole32 \
		-static-libstdc++ -static-libgcc \
    -o devcon.exe


WORKDIR /devcon
RUN i686-w64-mingw32-windmc msg.mc

WORKDIR /devcon/build-i686
RUN i686-w64-mingw32-windres ../devcon.rc rc.so
RUN i686-w64-mingw32-g++ -municode -Wno-write-strings \
    -DWIN32_LEAN_AND_MEAN=1 -DUNICODE -D_UNICODE \
    ../*.cpp rc.so \
    -lsetupapi -lole32 \
		-static-libstdc++ -static-libgcc \
    -o devcon.exe


