class CursorManager {
    constructor(editorContainer, editor) {
        this.editorContainer = editorContainer;
        this.editor = editor;
        this.cursors = {};
        this.cursorPositions = new Map();
        this.cursorsContainer = document.getElementById('cursors');
    }

    updateCursor(userId, position, color) {
        let cursor = this.cursors[userId];
        
        if (!cursor) {
            cursor = this.createCursor(userId, color);
            this.cursors[userId] = cursor;
        }
        
        // Store position for later use
        this.cursorPositions.set(userId, position);
        
        // Calculate and update cursor position
        const coords = this.getTextPositionInTextarea(position);
        
        cursor.style.left = coords.left + 'px';
        cursor.style.top = coords.top + 'px';
        cursor.style.color = color; // Set color for the ::before element
        
        // Debug log
        console.log(`Cursor updated for ${userId} at position ${position}:`, coords);
    }

    createCursor(userId, color) {
        const cursor = document.createElement('div');
        cursor.className = 'cursor';
        
        // Add special class for own cursor
        if (userId === this.currentUserId) {
            cursor.classList.add('own-cursor');
        }
        
        const label = document.createElement('div');
        label.className = 'cursor-label';
        label.textContent = userId;
        label.style.backgroundColor = color;
        label.style.color = 'white';
        cursor.appendChild(label);
        
        this.cursorsContainer.appendChild(cursor);
        return cursor;
    }
    
    setCurrentUserId(userId) {
        this.currentUserId = userId;
    }

    getTextPositionInTextarea(charIndex) {
        // Create a hidden div with the same styling as the textarea
        const mirror = document.createElement('div');
        const styles = window.getComputedStyle(this.editor);
        
        // Copy textarea styles
        mirror.style.cssText = `
            position: absolute;
            top: ${this.editor.offsetTop}px;
            left: ${this.editor.offsetLeft}px;
            width: ${styles.width};
            height: ${styles.height};
            padding: ${styles.padding};
            border: ${styles.border};
            font: ${styles.font};
            line-height: ${styles.lineHeight};
            white-space: pre-wrap;
            word-wrap: break-word;
            visibility: hidden;
            overflow: hidden;
        `;
        
        // Insert text up to cursor position
        const textBeforeCursor = this.editor.value.substring(0, charIndex);
        const textNode = document.createTextNode(textBeforeCursor);
        mirror.appendChild(textNode);
        
        // Add cursor marker
        const marker = document.createElement('span');
        marker.textContent = '|';
        marker.style.position = 'relative';
        mirror.appendChild(marker);
        
        // Add mirror to container (not body) for accurate positioning
        this.editorContainer.appendChild(mirror);
        
        // Get marker position relative to container
        const markerRect = marker.getBoundingClientRect();
        const containerRect = this.editorContainer.getBoundingClientRect();
        
        // Clean up
        this.editorContainer.removeChild(mirror);
        
        // Calculate final position accounting for scroll
        const x = markerRect.left - containerRect.left - this.editor.scrollLeft;
        const y = markerRect.top - containerRect.top - this.editor.scrollTop;
        
        console.log(`Position calculation for index ${charIndex}: x=${x}, y=${y}`);
        
        return {
            left: x,
            top: y
        };
    }

    removeCursor(userId) {
        if (this.cursors[userId]) {
            this.cursors[userId].remove();
            delete this.cursors[userId];
        }
        this.cursorPositions.delete(userId);
    }

    adjustCursorPosition(oldText, newText, cursorPos, changePos) {
        if (changePos >= cursorPos) {
            return cursorPos;
        }
        
        const diff = newText.length - oldText.length;
        return Math.max(0, cursorPos + diff);
    }

    adjustAllCursors(oldText, newText, changingUserId, connectedUsers) {
        this.cursorPositions.forEach((pos, uid) => {
            if (uid !== changingUserId) {
                const changePos = this.cursorPositions.get(changingUserId) || 0;
                const newPos = this.adjustCursorPosition(oldText, newText, pos, changePos);
                this.cursorPositions.set(uid, newPos);
                
                const color = connectedUsers.get(uid);
                if (color) {
                    this.updateCursor(uid, newPos, color);
                }
            }
        });
    }

    refreshAllPositions(connectedUsers) {
        this.cursorPositions.forEach((pos, uid) => {
            const color = connectedUsers.get(uid);
            if (color) {
                this.updateCursor(uid, pos, color);
            }
        });
    }

    clear() {
        Object.keys(this.cursors).forEach(userId => this.removeCursor(userId));
        this.cursorPositions.clear();
    }
}