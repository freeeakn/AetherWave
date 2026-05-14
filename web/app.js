const state = {
    connected: false,
    nodeAddress: '',
    encryptionKey: '',
    username: '',
};

const elements = {
    connectBtn: document.getElementById('connect-btn'),
    nodeAddress: document.getElementById('node-address'),
    encryptionKey: document.getElementById('encryption-key'),
    username: document.getElementById('username'),
    sendForm: document.getElementById('send-form'),
    recipient: document.getElementById('recipient'),
    message: document.getElementById('message'),
    messages: document.getElementById('messages'),
    refreshMessages: document.getElementById('refresh-messages'),
    refreshBlockchain: document.getElementById('refresh-blockchain'),
    refreshPeers: document.getElementById('refresh-peers'),
    blockchainInfo: document.getElementById('blockchain-info'),
    peersList: document.getElementById('peers-list'),
};

function escapeHTML(str) {
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', () => {
    elements.connectBtn.addEventListener('click', handleConnect);
    elements.sendForm.addEventListener('submit', handleSendMessage);
    elements.refreshMessages.addEventListener('click', fetchMessages);
    elements.refreshBlockchain.addEventListener('click', fetchBlockchainInfo);
    elements.refreshPeers.addEventListener('click', fetchPeers);

    const savedNodeAddress = localStorage.getItem('nodeAddress');
    const savedEncryptionKey = localStorage.getItem('encryptionKey');
    const savedUsername = localStorage.getItem('username');

    if (savedNodeAddress && savedEncryptionKey && savedUsername) {
        elements.nodeAddress.value = savedNodeAddress;
        elements.encryptionKey.value = savedEncryptionKey;
        elements.username.value = savedUsername;
        handleConnect();
    }
});

async function handleConnect() {
    const nodeAddress = elements.nodeAddress.value.trim();
    const encryptionKey = elements.encryptionKey.value.trim();
    const username = elements.username.value.trim();

    if (!nodeAddress) {
        showAlert('Error', 'Enter node address', 'danger');
        return;
    }

    if (!username) {
        showAlert('Error', 'Enter username', 'danger');
        return;
    }

    let key = encryptionKey;
    if (!encryptionKey) {
        try {
            key = await generateEncryptionKey();
            elements.encryptionKey.value = key;
            showAlert('Info', `Generated encryption key`, 'info');
        } catch (error) {
            showAlert('Error', `Failed to generate key: ${error.message}`, 'danger');
            return;
        }
    }

    try {
        elements.connectBtn.disabled = true;
        elements.connectBtn.textContent = 'Connecting...';

        state.nodeAddress = nodeAddress;
        state.encryptionKey = key;
        state.username = username;

        localStorage.setItem('nodeAddress', nodeAddress);
        localStorage.setItem('encryptionKey', key);
        localStorage.setItem('username', username);

        await fetchBlockchainInfo();

        state.connected = true;
        elements.connectBtn.textContent = 'Connected';
        elements.connectBtn.classList.remove('btn-primary');
        elements.connectBtn.classList.add('btn-success');

        await Promise.all([
            fetchMessages(),
            fetchPeers()
        ]);

        showAlert('Success', `Connected to ${nodeAddress}`, 'success');
    } catch (error) {
        showAlert('Connection error', error.message, 'danger');
        elements.connectBtn.textContent = 'Connect';
        elements.connectBtn.disabled = false;
    }
}

async function handleSendMessage(event) {
    event.preventDefault();

    if (!state.connected) {
        showAlert('Error', 'Connect to a node first', 'warning');
        return;
    }

    const recipient = elements.recipient.value.trim();
    const messageText = elements.message.value.trim();

    if (!recipient || !messageText) {
        showAlert('Error', 'Fill all fields', 'warning');
        return;
    }

    try {
        const submitBtn = elements.sendForm.querySelector('button[type="submit"]');
        submitBtn.disabled = true;
        submitBtn.textContent = 'Sending...';

        const response = await fetch(`http://${state.nodeAddress}/api/message`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                sender: state.username,
                recipient: recipient,
                content: messageText,
                key: state.encryptionKey
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();

        elements.message.value = '';

        await Promise.all([
            fetchMessages(),
            fetchBlockchainInfo()
        ]);

        showAlert('Success', 'Message sent and added to blockchain', 'success');
    } catch (error) {
        showAlert('Send error', error.message, 'danger');
    } finally {
        const submitBtn = elements.sendForm.querySelector('button[type="submit"]');
        submitBtn.disabled = false;
        submitBtn.textContent = 'Send';
    }
}

async function fetchMessages() {
    if (!state.connected) {
        showAlert('Error', 'Connect to a node first', 'warning');
        return;
    }

    try {
        elements.refreshMessages.disabled = true;
        elements.messages.innerHTML = '<div class="text-center"><div class="spinner-border" role="status"><span class="visually-hidden">Loading...</span></div></div>';

        const response = await fetch(`http://${state.nodeAddress}/api/messages`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                username: state.username,
                key: state.encryptionKey
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const messages = await response.json();

        if (messages.length === 0) {
            elements.messages.innerHTML = '<div class="alert alert-info">No messages</div>';
        } else {
            elements.messages.innerHTML = '';
            messages.forEach(msg => {
                const isOutgoing = state.connected && msg.sender === state.username;
                const messageEl = document.createElement('div');
                messageEl.className = isOutgoing ? 'message-item outgoing' : 'message-item';
                messageEl.appendChild(buildMessageContent(msg, isOutgoing));
                elements.messages.appendChild(messageEl);
            });
        }
    } catch (error) {
        showAlert('Messages error', error.message, 'danger');
        elements.messages.innerHTML = '<div class="alert alert-danger">Failed to load messages</div>';
    } finally {
        elements.refreshMessages.disabled = false;
    }
}

function buildMessageContent(msg, isOutgoing) {
    const container = document.createElement('div');
    container.className = 'd-flex justify-content-between';

    const sender = document.createElement('strong');
    sender.textContent = isOutgoing ? `You -> ${msg.recipient}` : msg.sender;

    const time = document.createElement('small');
    time.className = 'text-muted';
    time.textContent = formatTimestamp(msg.timestamp);

    container.appendChild(sender);
    container.appendChild(time);

    const body = document.createElement('p');
    body.className = 'mb-0';
    body.textContent = msg.content;

    const wrapper = document.createElement('div');
    wrapper.appendChild(container);
    wrapper.appendChild(body);

    return wrapper;
}

async function fetchBlockchainInfo() {
    try {
        elements.refreshBlockchain.disabled = true;
        elements.blockchainInfo.innerHTML = '<div class="text-center"><div class="spinner-border" role="status"><span class="visually-hidden">Loading...</span></div></div>';

        const response = await fetch(`http://${state.nodeAddress}/api/blockchain`);

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        elements.blockchainInfo.innerHTML = `
            <div class="row text-start">
                <div class="col-md-4">
                    <h5>Overview</h5>
                    <p><strong>Blocks:</strong> ${escapeHTML(String(data.blockCount))}</p>
                    <p><strong>Difficulty:</strong> ${escapeHTML(String(data.difficulty))}</p>
                    <p><strong>Messages:</strong> ${escapeHTML(String(data.messageCount))}</p>
                </div>
                <div class="col-md-8">
                    <h5>Last block</h5>
                    <p><strong>Index:</strong> ${escapeHTML(String(data.lastBlock.index))}</p>
                    <p><strong>Hash:</strong> <code>${escapeHTML(data.lastBlock.hash)}</code></p>
                    <p><strong>Prev hash:</strong> <code>${escapeHTML(data.lastBlock.prevHash)}</code></p>
                    <p><strong>Nonce:</strong> ${escapeHTML(String(data.lastBlock.nonce))}</p>
                    <p><strong>Timestamp:</strong> ${escapeHTML(formatTimestamp(data.lastBlock.timestamp))}</p>
                </div>
            </div>
        `;
    } catch (error) {
        showAlert('Blockchain error', error.message, 'danger');
        elements.blockchainInfo.innerHTML = '<div class="alert alert-danger">Failed to load blockchain info</div>';
    } finally {
        elements.refreshBlockchain.disabled = false;
    }
}

async function fetchPeers() {
    if (!state.connected) {
        return;
    }

    try {
        elements.refreshPeers.disabled = true;
        elements.peersList.innerHTML = '<div class="text-center"><div class="spinner-border" role="status"><span class="visually-hidden">Loading...</span></div></div>';

        const response = await fetch(`http://${state.nodeAddress}/api/peers`);

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const peers = await response.json();

        if (peers.length === 0) {
            elements.peersList.innerHTML = '<div class="alert alert-info">No peers</div>';
        } else {
            let html = '<div class="table-responsive"><table class="table table-striped"><thead><tr><th>Address</th><th>Status</th><th>Last seen</th></tr></thead><tbody>';
            for (const peer of peers) {
                html += `<tr><td>${escapeHTML(peer.address)}</td><td><span class="badge ${peer.active ? 'bg-success' : 'bg-danger'}">${peer.active ? 'Active' : 'Inactive'}</span></td><td>${escapeHTML(formatTimestamp(peer.lastSeen))}</td></tr>`;
            }
            html += '</tbody></table></div>';
            elements.peersList.innerHTML = html;
        }
    } catch (error) {
        showAlert('Peers error', error.message, 'danger');
        elements.peersList.innerHTML = '<div class="alert alert-danger">Failed to load peers</div>';
    } finally {
        elements.refreshPeers.disabled = false;
    }
}

function formatTimestamp(timestamp) {
    try {
        const date = new Date(timestamp * 1000);
        return date.toLocaleString();
    } catch (e) {
        return String(timestamp);
    }
}

async function generateEncryptionKey() {
    const array = new Uint8Array(32);
    window.crypto.getRandomValues(array);
    return Array.from(array)
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}

function showAlert(title, message, type = 'info') {
    const alertContainer = document.createElement('div');
    alertContainer.className = 'container mt-3';

    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
    alertDiv.setAttribute('role', 'alert');

    const strong = document.createElement('strong');
    strong.textContent = title;
    alertDiv.appendChild(strong);

    alertDiv.appendChild(document.createTextNode(' ' + message));

    const closeBtn = document.createElement('button');
    closeBtn.type = 'button';
    closeBtn.className = 'btn-close';
    closeBtn.setAttribute('data-bs-dismiss', 'alert');
    closeBtn.setAttribute('aria-label', 'Close');
    alertDiv.appendChild(closeBtn);

    alertContainer.appendChild(alertDiv);
    document.body.insertBefore(alertContainer, document.body.firstChild);

    setTimeout(() => {
        alertContainer.remove();
    }, 5000);
}
