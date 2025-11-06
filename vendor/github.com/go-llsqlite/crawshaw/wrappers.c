/*
  Copyright (c) 2020 David Crawshaw <david@zentus.com>

  Permission to use, copy, modify, and distribute this software for any
  purpose with or without fee is hereby granted, provided that the above
  copyright notice and this permission notice appear in all copies.

  THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
  WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
  MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
  ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
  WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
  ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
  OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

#include <stdint.h>
#include <stdlib.h>
#include <sqlite3.h>

void cfree(void *p) {
        free(p);
};

extern int go_strm_w_tramp(uintptr_t, char*, int);
int c_strm_w_tramp(void *pOut, const void *pData, int n) {
        return go_strm_w_tramp((uintptr_t)pOut, (char*)pData, n);
}

extern int go_strm_r_tramp(uintptr_t, char*, int*);
int c_strm_r_tramp(void *pOut, const void *pData, int *pN) {
        return go_strm_r_tramp((uintptr_t)pOut, (char*)pData, pN);
}

extern int go_xapply_conflict_tramp(uintptr_t, int, sqlite3_changeset_iter*);
int c_xapply_conflict_tramp(void* pCtx, int eConflict, sqlite3_changeset_iter* p) {
        return go_xapply_conflict_tramp((uintptr_t)pCtx, eConflict, p);
}

extern int go_xapply_filter_tramp(uintptr_t, char*);
int c_xapply_filter_tramp(void* pCtx, const char* zTab) {
        return go_xapply_filter_tramp((uintptr_t)pCtx, (char*)zTab);
}

extern void go_func_tramp(sqlite3_context*, int, sqlite3_value**);
void c_func_tramp(sqlite3_context* ctx, int n, sqlite3_value** valarray) {
        return go_func_tramp(ctx, n, valarray);
}

extern void go_step_tramp(sqlite3_context*, int, sqlite3_value**);
void c_step_tramp(sqlite3_context* ctx, int n, sqlite3_value** valarray) {
        return go_step_tramp(ctx, n, valarray);
}

extern void go_final_tramp(sqlite3_context*);
void c_final_tramp(sqlite3_context* ctx) {
        return go_final_tramp(ctx);
}

extern void go_destroy_tramp(uintptr_t);
void c_destroy_tramp(void* ptr) {
        return go_destroy_tramp((uintptr_t)ptr);
}

extern int go_sqlite_auth_tramp(uintptr_t, int, char*, char*, char*, char*);
int c_auth_tramp(void *userData, int action, const char* arg1, const char* arg2, const char* db, const char* trigger) {
        return go_sqlite_auth_tramp((uintptr_t)userData, action, (char*)arg1, (char*)arg2, (char*)db, (char*)trigger);
}

extern void go_log_fn(void*, int, char*);
void c_log_fn(void* pArg, int code, char* msg) {
        return go_log_fn(pArg, code, msg);
}

extern int goBusyHandlerCallback(void *, int);
int c_goBusyHandlerCallback(void *pArg, int count) {
        return goBusyHandlerCallback(pArg, count);
}