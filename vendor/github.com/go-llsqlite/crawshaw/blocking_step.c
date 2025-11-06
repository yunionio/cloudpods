// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// This file defines the wait_for_unlock_notify function.
// See the documentation on Stmt.Step.

#include <blocking_step.h>
#include <stdlib.h>

unlock_note* unlock_note_alloc() {
	unlock_note* un = (unlock_note*)malloc(sizeof(unlock_note));
	pthread_mutex_init(&un->mu, 0);
	pthread_cond_init(&un->cond, 0);
	return un;
}

void unlock_note_free(unlock_note* un) {
	pthread_cond_destroy(&un->cond);
	pthread_mutex_destroy(&un->mu);
	free(un);
}

void unlock_note_fire(unlock_note* un) {
	pthread_mutex_lock(&un->mu);
	un->fired = 1;
	pthread_cond_signal(&un->cond);
	pthread_mutex_unlock(&un->mu);
}

static void unlock_notify_cb(void **apArg, int nArg) {
	for(int i=0; i < nArg; i++) {
		unlock_note_fire((unlock_note*)apArg[i]);
	}
}

int wait_for_unlock_notify(sqlite3 *db, unlock_note* un) {
	un->fired = 0;

	int res = sqlite3_unlock_notify(db, unlock_notify_cb, (void *)un);

	if (res == SQLITE_OK) {
		pthread_mutex_lock(&un->mu);
		if (!un->fired) {
			pthread_cond_wait(&un->cond, &un->mu);
		}
		pthread_mutex_unlock(&un->mu);
	}

	return res;
}
