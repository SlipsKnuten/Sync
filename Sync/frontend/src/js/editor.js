class Editor {
    constructor(editorElement, onUpdate, onCursorMove) {
        this.editor = editorElement;
        this.onUpdate = onUpdate;
        this.onCursorMove = onCursorMove;
        this.isRemoteUpdate = false;
        this.saveTimer = null;
        this.lastSavedContent = '';
        
        this.setupEventListeners();
        this.setupAutoSave();
    }

    setupEventListeners() {
        let hasUnsavedChanges = false;
        
        this.editor.addEventListener('input', () => {
            if (!this.isRemoteUpdate && this.onUpdate) {
                this.onUpdate(this.editor.value);
                this.scheduleSave();
                
                // Mark that we have unsaved changes
                hasUnsavedChanges = true;
                
                // Save immediately on first change if user is authenticated
                if (!this.lastSavedContent && localStorage.getItem('token')) {
                    setTimeout(() => this.saveNow(), 1000);
                }
            }
        });

        // Send cursor position on various events
        ['keyup', 'keydown', 'click', 'select', 'focus'].forEach(eventType => {
            this.editor.addEventListener(eventType, () => {
                if (this.onCursorMove) {
                    this.onCursorMove(this.editor.selectionStart);
                }
            });
        });
        
        // Also send cursor position when using arrow keys or page up/down
        this.editor.addEventListener('keydown', (e) => {
            const navigationKeys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 
                                   'PageUp', 'PageDown', 'Home', 'End'];
            if (navigationKeys.includes(e.key)) {
                setTimeout(() => {
                    if (this.onCursorMove) {
                        this.onCursorMove(this.editor.selectionStart);
                    }
                }, 0);
            }
        });
    }

    setupAutoSave() {
        // Save on page unload
        window.addEventListener('beforeunload', (e) => {
            if (this.editor.value !== this.lastSavedContent) {
                // Try to save synchronously before leaving
                this.saveNowSync();
            }
        });

        // Also save when clicking any link that leaves the page
        document.addEventListener('click', (e) => {
            const link = e.target.closest('a');
            if (link && link.href && !link.href.includes('#')) {
                this.saveNow();
            }
        });

        // Periodic save every 30 seconds if there are changes
        setInterval(() => {
            if (this.editor.value !== this.lastSavedContent) {
                this.saveNow();
            }
        }, 30000);
    }

    scheduleSave() {
        // Clear existing timer
        if (this.saveTimer) {
            clearTimeout(this.saveTimer);
        }

        // Schedule save after 5 seconds of inactivity
        this.saveTimer = setTimeout(() => {
            this.saveNow();
        }, 5000);
    }

    async saveNow() {
        const content = this.editor.value;
        
        // Don't save if content hasn't changed
        if (content === this.lastSavedContent) {
            return;
        }

        // Get session code from URL
        const urlParams = new URLSearchParams(window.location.search);
        const sessionCode = urlParams.get('session');
        
        if (!sessionCode) {
            return;
        }

        // Check if user is authenticated
        const token = localStorage.getItem('token');
        
        console.log('Attempting to save document...', {
            sessionCode,
            hasToken: !!token,
            contentLength: content.length
        });
        
        try {
            const headers = {
                'Content-Type': 'application/json',
            };

            // Add auth header if user is logged in
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }

            const response = await fetch('http://localhost:8080/api/document/save', {
                method: 'POST',
                headers: headers,
                body: JSON.stringify({
                    session_code: sessionCode,
                    content: content
                })
            });

            const result = await response.json();
            
            if (response.ok) {
                this.lastSavedContent = content;
                console.log('Document saved successfully:', result);
                this.showSaveIndicator('Saved', 'success');
            } else {
                console.error('Failed to save document:', result);
                this.showSaveIndicator('Save failed', 'error');
            }
        } catch (error) {
            console.error('Error saving document:', error);
            this.showSaveIndicator('Save failed', 'error');
        }
    }

    // Synchronous save for beforeunload
    saveNowSync() {
        const content = this.editor.value;
        
        // Don't save if content hasn't changed
        if (content === this.lastSavedContent) {
            return;
        }

        // Get session code from URL
        const urlParams = new URLSearchParams(window.location.search);
        const sessionCode = urlParams.get('session');
        
        if (!sessionCode) {
            return;
        }

        // Check if user is authenticated
        const token = localStorage.getItem('token');
        
        try {
            const headers = {
                'Content-Type': 'application/json',
            };

            // Add auth header if user is logged in
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }

            // Use sendBeacon for reliable delivery on page unload
            const data = JSON.stringify({
                session_code: sessionCode,
                content: content
            });

            navigator.sendBeacon('http://localhost:8080/api/document/save', 
                new Blob([data], { type: 'application/json' }));
            
            this.lastSavedContent = content;
        } catch (error) {
            console.error('Error saving document:', error);
        }
    }

    showSaveIndicator(message, type) {
        // Get or create save indicator
        let indicator = document.getElementById('saveIndicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'saveIndicator';
            indicator.style.cssText = `
                position: fixed;
                bottom: 20px;
                right: 20px;
                padding: 10px 20px;
                border-radius: 5px;
                font-size: 14px;
                transition: opacity 0.3s;
                z-index: 1000;
            `;
            document.body.appendChild(indicator);
        }

        // Set style based on type
        if (type === 'success') {
            indicator.style.backgroundColor = '#10b981';
            indicator.style.color = 'white';
        } else {
            indicator.style.backgroundColor = '#ef4444';
            indicator.style.color = 'white';
        }

        indicator.textContent = message;
        indicator.style.opacity = '1';

        // Hide after 2 seconds
        setTimeout(() => {
            indicator.style.opacity = '0';
        }, 2000);
    }

    updateContent(content, preserveCursor = true) {
        this.isRemoteUpdate = true;
        const cursorPos = preserveCursor ? this.editor.selectionStart : 0;
        this.editor.value = content;
        if (preserveCursor) {
            this.editor.setSelectionRange(cursorPos, cursorPos);
        }
        this.isRemoteUpdate = false;
        this.lastSavedContent = content;
    }

    getCursorPosition() {
        return this.editor.selectionStart;
    }

    setCursorPosition(position) {
        this.editor.setSelectionRange(position, position);
    }

    focus() {
        this.editor.focus();
    }
}