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

#ifndef WRAPPERS_H
#define WRAPPERS_H

/* cfree wraps free to fix https://github.com/crawshaw/sqlite/issues/60 */
void cfree(void *p);

int c_strm_w_tramp(void*, const void*, int);
int c_strm_r_tramp(void*, const void*, int*);

int c_xapply_conflict_tramp(void*, int, sqlite3_changeset_iter*);
int c_xapply_filter_tramp(void*, const char*);

void c_log_fn(void*, int, char*);
int c_auth_tramp(void*, int, const char*, const char*, const char*, const char*);

void c_func_tramp(sqlite3_context*, int, sqlite3_value**);
void c_step_tramp(sqlite3_context*, int, sqlite3_value**);
void c_final_tramp(sqlite3_context*);
void c_destroy_tramp(void*);

int c_goBusyHandlerCallback(void*, int);

#endif // WRAPPERS_H
