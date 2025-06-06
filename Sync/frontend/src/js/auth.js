// Authentication functions
const API_BASE = 'http://localhost:8080/api';

// Check if user is authenticated
function checkAuth() {
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('user');
    
    // Only update UI elements if they exist (for pages that have them)
    const authButtons = document.getElementById('authButtons');
    const userInfo = document.getElementById('userInfo');
    const username = document.getElementById('username');
    
    if (token && user) {
        const userData = JSON.parse(user);
        if (authButtons) authButtons.classList.add('hidden');
        if (userInfo) {
            userInfo.classList.remove('hidden');
            userInfo.classList.add('flex');
        }
        if (username) username.textContent = userData.username;
    } else {
        if (authButtons) authButtons.classList.remove('hidden');
        if (userInfo) userInfo.classList.add('hidden');
    }
}

// Modal functions
function showLoginModal() {
    const loginModal = document.getElementById('loginModal');
    if (loginModal) {
        loginModal.classList.remove('hidden');
        loginModal.classList.add('flex');
    }
}

function hideLoginModal() {
    const loginModal = document.getElementById('loginModal');
    const loginError = document.getElementById('loginError');
    const loginForm = document.getElementById('loginForm');
    
    if (loginModal) loginModal.classList.add('hidden');
    if (loginError) loginError.classList.add('hidden');
    if (loginForm) loginForm.reset();
}

function showRegisterModal() {
    const registerModal = document.getElementById('registerModal');
    if (registerModal) {
        registerModal.classList.remove('hidden');
        registerModal.classList.add('flex');
    }
}

function hideRegisterModal() {
    const registerModal = document.getElementById('registerModal');
    const registerError = document.getElementById('registerError');
    const registerForm = document.getElementById('registerForm');
    
    if (registerModal) registerModal.classList.add('hidden');
    if (registerError) registerError.classList.add('hidden');
    if (registerForm) registerForm.reset();
}

function showSessionsModal() {
    const sessionsModal = document.getElementById('sessionsModal');
    if (sessionsModal) {
        sessionsModal.classList.remove('hidden');
        sessionsModal.classList.add('flex');
    }
}

function hideSessionsModal() {
    const sessionsModal = document.getElementById('sessionsModal');
    if (sessionsModal) sessionsModal.classList.add('hidden');
}

// Login form handler
document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    
    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });
        
        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('token', data.token);
            localStorage.setItem('user', JSON.stringify(data.user));
            hideLoginModal();
            checkAuth();
        } else {
            const error = await response.text();
            document.getElementById('loginError').textContent = error;
            document.getElementById('loginError').classList.remove('hidden');
        }
    } catch (error) {
        document.getElementById('loginError').textContent = 'An error occurred. Please try again.';
        document.getElementById('loginError').classList.remove('hidden');
    }
});

// Register form handler
document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const username = document.getElementById('registerUsername').value;
    const email = document.getElementById('registerEmail').value;
    const password = document.getElementById('registerPassword').value;
    
    try {
        const response = await fetch(`${API_BASE}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, email, password }),
        });
        
        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('token', data.token);
            localStorage.setItem('user', JSON.stringify(data.user));
            hideRegisterModal();
            checkAuth();
        } else {
            const error = await response.text();
            document.getElementById('registerError').textContent = error;
            document.getElementById('registerError').classList.remove('hidden');
        }
    } catch (error) {
        document.getElementById('registerError').textContent = 'An error occurred. Please try again.';
        document.getElementById('registerError').classList.remove('hidden');
    }
});

// Logout function
function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    checkAuth();
}

// View sessions function
async function viewSessions() {
    const token = localStorage.getItem('token');
    
    if (!token) {
        showLoginModal();
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/sessions`, {
            headers: {
                'Authorization': `Bearer ${token}`,
            },
        });
        
        if (response.ok) {
            const sessions = await response.json();
            displaySessions(sessions);
            showSessionsModal();
        } else if (response.status === 401) {
            logout();
            showLoginModal();
        }
    } catch (error) {
        console.error('Failed to load sessions:', error);
    }
}

// Display sessions in modal
function displaySessions(sessions) {
    const sessionsList = document.getElementById('sessionsList');
    
    if (sessions.length === 0) {
        sessionsList.innerHTML = '<p class="text-gray-500">No sessions found. Create a new session to get started!</p>';
        return;
    }
    
    sessionsList.innerHTML = sessions.map(session => `
        <div class="border rounded-lg p-4 hover:bg-gray-50 cursor-pointer" onclick="window.location.href='editor.html?session=${session.session_code}'">
            <div class="flex justify-between items-center">
                <div>
                    <h3 class="font-semibold">Session: ${session.session_code}</h3>
                    <p class="text-sm text-gray-600">Last modified: ${new Date(session.last_modified).toLocaleString()}</p>
                </div>
                <button class="text-blue-600 hover:text-blue-800">Open →</button>
            </div>
        </div>
    `).join('');
}