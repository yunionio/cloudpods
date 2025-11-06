// This file declares the wait_for_unlock_notify function.
// See the documentation on Stmt.Step.

#include <sqlite3.h>
#include <pthread.h>

typedef struct unlock_note {
	int fired;
	pthread_cond_t cond;
	pthread_mutex_t mu;
} unlock_note;

unlock_note* unlock_note_alloc();
void unlock_note_fire(unlock_note* un);
void unlock_note_free(unlock_note* un);

int wait_for_unlock_notify(sqlite3 *db, unlock_note* un);
