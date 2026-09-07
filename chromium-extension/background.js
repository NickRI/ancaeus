// background.js
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (msg.type !== 'geo') return;
    fetch('__SERVER_URL__', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ considerIp: true })
    })
        .then(r => {
            if (!r.ok) {
                return r.json().then(err => Promise.reject(new Error(err.error?.message || `HTTP ${r.status}`)));
            }
            return r.json();
        })
        .then(data => sendResponse({
            coords: { latitude: data.location.lat, longitude: data.location.lng, accuracy: data.accuracy || 0 },
            timestamp: Date.now()
        }))
        .catch(e => sendResponse({ error: e.message }));
    return true;
});
