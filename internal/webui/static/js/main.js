const clientID = Math.random().toString(36).substr(2, 9);

function pollUpdates() {
    fetch(`/api/updates?id=${clientID}`)
        .then(response => {
            if (response.status === 204) {
                // No content, poll again
                pollUpdates();
                return;
            }
            return response.json();
        })
        .then(data => {
            if (data) {
                updateUI(data);
            }
            pollUpdates();
        })
        .catch(() => {
            setTimeout(pollUpdates, 5000); // Retry after 5 seconds on error
        });
}

// Start polling
pollUpdates();

function updateUI(data) {
    if (data.peers) {
        updatePeersList(data.peers);
    }
    if (data.files) {
        updateFilesList(data.files);
    }
}

function updatePeersList(peers) {
    const peersList = document.getElementById('peers-list');
    peersList.innerHTML = peers.map(peer => `
        <div class="peer">
            <span>${peer.address}</span>
            <span>${peer.lastSeen}</span>
        </div>
    `).join('');
}

function updateFilesList(files) {
    const filesList = document.getElementById('files-list');
    filesList.innerHTML = files.map(file => `
        <div class="file">
            <span>${file.name}</span>
            <span>${file.size}</span>
            <button onclick="downloadFile('${file.id}')">Download</button>
        </div>
    `).join('');
}

function uploadFile() {
    const fileInput = document.getElementById('file-input');
    const file = fileInput.files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('file', file);

    fetch('/api/files', {
        method: 'POST',
        body: formData
    });
}

function downloadFile(fileId) {
    window.location.href = `/api/files/${fileId}/download`;
}
