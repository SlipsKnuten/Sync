// Initialize the collaborative editor
(function() {
    // Get session code from URL
    const urlParams = new URLSearchParams(window.location.search);
    const sessionCode = urlParams.get('session');
    
    // Redirect to landing page if no session code
    if (!sessionCode) {
        window.location.href = 'index.html';
        return;
    }
    
    // Display session code
    document.getElementById('sessionCode').textContent = sessionCode;
    
    // Copy session code functionality
    document.getElementById('copyCode').addEventListener('click', () => {
        navigator.clipboard.writeText(sessionCode).then(() => {
            const btn = document.getElementById('copyCode');
            const originalText = btn.textContent;
            btn.textContent = 'Copied!';
            btn.classList.add('bg-green-200');
            setTimeout(() => {
                btn.textContent = originalText;
                btn.classList.remove('bg-green-200');
            }, 2000);
        });
    });
    
    const userId = 'user' + Math.random().toString(36).substr(2, 9);
    const connectedUsers = new Map();
    let cursorManager;
    let editor;
    let wsManager;

    // DOM elements
    const editorElement = document.getElementById('editor');
    const editorContainer = document.getElementById('editor-container');
    const statusEl = document.getElementById('status');
    const usersEl = document.getElementById('users');

    // Initialize components
    function init() {
        // Create cursor manager
        cursorManager = new CursorManager(editorContainer, editorElement);
        cursorManager.setCurrentUserId(userId);

        // Create editor
        editor = new Editor(
            editorElement,
            (content) => {
                wsManager.sendMessage('update', { content });
                // Send cursor position after content update
                const pos = editor.getCursorPosition();
                cursorManager.cursorPositions.set(userId, pos);
                wsManager.sendMessage('cursor', { cursorPos: pos });
            },
            (position) => {
                cursorManager.cursorPositions.set(userId, position);
                wsManager.sendMessage('cursor', { cursorPos: position });
                // Update own cursor display
                const myColor = connectedUsers.get(userId);
                if (myColor) {
                    cursorManager.updateCursor(userId, position, myColor);
                }
            }
        );

        // Create WebSocket manager
        wsManager = new WebSocketManager(
            userId,
            sessionCode,
            handleMessage,
            handleStatusChange
        );

        // Setup scroll handler
        editorElement.addEventListener('scroll', () => {
            cursorManager.refreshAllPositions(connectedUsers);
        });

        // Connect to server
        wsManager.connect();
    }

    // Handle incoming messages
    function handleMessage(msg) {
        switch(msg.type) {
            case 'init':
                editor.updateContent(msg.content, false);
                updateUserBadge(msg.userId, msg.color, true);
                connectedUsers.set(msg.userId, msg.color);
                // Show own cursor
                cursorManager.updateCursor(msg.userId, 0, msg.color);
                break;
            
            case 'update':
                if (msg.userId !== userId) {
                    const oldContent = editorElement.value;
                    const myCursorPos = editor.getCursorPosition();
                    
                    editor.updateContent(msg.content, false);
                    
                    // Adjust cursor positions
                    const changePos = cursorManager.cursorPositions.get(msg.userId) || 0;
                    const newCursorPos = cursorManager.adjustCursorPosition(
                        oldContent, 
                        msg.content, 
                        myCursorPos, 
                        changePos
                    );
                    editor.setCursorPosition(newCursorPos);
                    
                    cursorManager.adjustAllCursors(oldContent, msg.content, msg.userId, connectedUsers);
                }
                break;
            
            case 'cursor':
                if (msg.userId !== userId) {
                    cursorManager.updateCursor(msg.userId, msg.cursorPos, msg.color);
                }
                break;
            
            case 'userJoined':
                if (msg.userId !== userId && !connectedUsers.has(msg.userId)) {
                    updateUserBadge(msg.userId, msg.color, false);
                    connectedUsers.set(msg.userId, msg.color);
                }
                break;
            
            case 'userLeft':
                cursorManager.removeCursor(msg.userId);
                removeUserBadge(msg.userId);
                connectedUsers.delete(msg.userId);
                break;
        }
    }

    // Handle connection status changes
    function handleStatusChange(status) {
        if (status === 'connected') {
            statusEl.textContent = 'Connected';
            statusEl.className = 'text-sm text-green-600';
        } else {
            statusEl.textContent = 'Disconnected - Reconnecting...';
            statusEl.className = 'text-sm text-red-600';
            // Clear all users except self
            connectedUsers.clear();
            usersEl.innerHTML = '';
            cursorManager.clear();
        }
    }

    // Update user badge
    function updateUserBadge(userId, color, isSelf) {
        if (document.getElementById(`user-${userId}`)) return;
        
        const badge = document.createElement('div');
        badge.id = `user-${userId}`;
        badge.className = 'px-3 py-1 rounded-full text-white text-sm';
        badge.style.backgroundColor = color;
        badge.textContent = isSelf ? `${userId} (You)` : userId;
        usersEl.appendChild(badge);
    }

    // Remove user badge
    function removeUserBadge(userId) {
        const badge = document.getElementById(`user-${userId}`);
        if (badge) badge.remove();
    }

    // Start the application
    init();
})();