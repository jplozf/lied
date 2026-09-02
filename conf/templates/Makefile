# /*****************************************************\
#  project  : termOS
#  filename : Makefile
#
#  Copyright (C) 2003 by Marcin Wi¶nios <aliceman@wp.pl>
#
#  See file COPYING for copyright details.
# \*****************************************************/

CC = gcc
CFLAGS = -g3 -Wall
# CFLAGS = -O3 -Wall -pedantic
LIBS = -lncurses
DISTNAME = termos-`date +"%d%m%Y"`

all: termos

# termos
termos_objects = termos.o menus.o print.o run.o aster.o go.o\
		 chgcmd.o crtcmd.o dspcmd.o wrkcmd.o \
		 call.o cd.o cpyf.o dltf.o dspcurdir.o dspwsusr.o md.o mov.o rd.o rnm.o wrkactjob.o

termos: $(termos_objects)
	$(CC) $(CFLAGS) -o termos $(termos_objects) $(LIBS)

$(termos_objects) = termos.h

# test
test: termos
	./termos

# distribution
dist: clean
	mkdir $(DISTNAME)
	cp *.c *.h Makefile AUTHOR BUGS COMMANDS COPYING README TODO $(DISTNAME)/
	tar zcvf $(DISTNAME).tar.gz $(DISTNAME)
	rm -rf $(DISTNAME)

# cleaning
clean:
	rm -f termos $(termos_objects) core

