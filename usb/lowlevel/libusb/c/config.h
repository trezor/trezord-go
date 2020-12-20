#ifndef CONFIG_H

#define CONFIG_H

#ifndef _MSC_VER

#define PRINTF_FORMAT(a, b) __attribute__ ((__format__ (__printf__, a, b)))

#else

#define PRINTF_FORMAT(a, b)

#endif

#endif
