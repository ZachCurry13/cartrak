// The last answer of every read-only screen, kept on this device in IndexedDB, so the app still shows its data with no signal.
// Nothing here is ever sent anywhere. It is wiped at sign-out, and the oldest answers go first once there are too many.
const DB = 'cartrak-offline', STORE = 'answers', MAX_ANSWERS = 400, MAX_BYTES = 1500000;
let dbp = null, saves = 0;

function open() {
  if (!dbp) {
    dbp = new Promise((resolve, reject) => {
      if (!('indexedDB' in window)) { reject(new Error('no storage')); return; }
      const rq = indexedDB.open(DB, 1);
      rq.onupgradeneeded = () => { const os = rq.result.createObjectStore(STORE, { keyPath: 'k' }); os.createIndex('t', 't'); };
      rq.onsuccess = () => resolve(rq.result);
      rq.onerror = () => reject(rq.error);
    });
    dbp.catch(() => { dbp = null; });
  }
  return dbp;
}

const done = (tx) => new Promise((resolve, reject) => { tx.oncomplete = () => resolve(); tx.onerror = tx.onabort = () => reject(tx.error); });

// Keeps one answer. Failures (private window, full storage) are ignored: the app works without it.
export async function saveAnswer(k, value) {
  try {
    if (JSON.stringify(value).length > MAX_BYTES) return;
    const db = await open();
    const tx = db.transaction(STORE, 'readwrite');
    tx.objectStore(STORE).put({ k, t: Date.now(), v: value });
    await done(tx);
    if (++saves % 25 === 0) await prune(db);
  } catch (e) { /* storage blocked or full */ }
}

// The kept answer for k as { t, v }, or null.
export async function loadAnswer(k) {
  try {
    const db = await open();
    return await new Promise((resolve, reject) => {
      const rq = db.transaction(STORE).objectStore(STORE).get(k);
      rq.onsuccess = () => resolve(rq.result || null);
      rq.onerror = () => reject(rq.error);
    });
  } catch (e) { return null; }
}

async function prune(db) {
  const os = db.transaction(STORE, 'readwrite');
  const store = os.objectStore(STORE);
  const n = await new Promise((resolve) => { const c = store.count(); c.onsuccess = () => resolve(c.result); c.onerror = () => resolve(0); });
  let extra = n - MAX_ANSWERS;
  if (extra > 0) {
    store.index('t').openCursor().onsuccess = (ev) => {
      const cur = ev.target.result;
      if (cur && extra-- > 0) { cur.delete(); cur.continue(); }
    };
  }
  await done(os);
}

// Everything kept on this device, answers and map pictures: used at sign-out.
export async function forgetOffline() {
  try { const db = await open(); const tx = db.transaction(STORE, 'readwrite'); tx.objectStore(STORE).clear(); await done(tx); } catch (e) { /* nothing kept */ }
  try { await caches.delete('cartrak-tiles'); } catch (e) { /* no cache storage */ }
}
