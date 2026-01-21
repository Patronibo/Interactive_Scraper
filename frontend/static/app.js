const API_BASE = '/api';
let currentPage = 1;
const pageSize = 20;
let currentEntryId = null;
let authToken = null;
let currentTab = 'overview';
let entriesTabPage = 1;
let criticalityTabPage = 1;

function showNotification(message, type = 'info') {
    console.log(`[NOTIFICATION ${type.toUpperCase()}]: ${message}`);
    
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        border-radius: 5px;
        color: white;
        font-weight: 500;
        z-index: 10000;
        max-width: 400px;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
        opacity: 0;
        transform: translateX(100%);
        transition: all 0.3s ease-out;
    `;
    
    if (type === 'success') {
        notification.style.background = '#4CAF50';
    } else if (type === 'error') {
        notification.style.background = '#f44336';
    } else {
        notification.style.background = '#2196F3';
    }
    
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.opacity = '1';
        notification.style.transform = 'translateX(0)';
    }, 10);
    
    setTimeout(() => {
        notification.style.opacity = '0';
        notification.style.transform = 'translateX(100%)';
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 300);
    }, 4000);
}

class MatrixRain {
    constructor(canvas) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.chars = '01アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン';
        this.fontSize = 14;
        this.columns = 0;
        this.drops = [];
        this.animationId = null;
        
        this.init();
        this.animate();
        
        window.addEventListener('resize', () => this.init());
    }
    
    init() {
        this.canvas.width = window.innerWidth;
        this.canvas.height = window.innerHeight;
        this.columns = Math.floor(this.canvas.width / this.fontSize);
        this.drops = [];
        
        for (let i = 0; i < this.columns; i++) {
            this.drops[i] = Math.random() * -100;
        }
    }
    
    draw() {
        this.ctx.fillStyle = 'rgba(0, 0, 0, 0.05)';
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
        
        this.ctx.font = `${this.fontSize}px monospace`;
        
        for (let i = 0; i < this.drops.length; i++) {
            const text = this.chars[Math.floor(Math.random() * this.chars.length)];
            const x = i * this.fontSize;
            const y = this.drops[i] * this.fontSize;
            
            const opacity = Math.max(0, 1 - (this.drops[i] / 50));
            this.ctx.fillStyle = `rgba(0, 255, 0, ${opacity})`;
            
            this.ctx.fillText(text, x, y);
            
            if (this.drops[i] * this.fontSize > this.canvas.height && Math.random() > 0.975) {
                this.drops[i] = 0;
            }
            
            this.drops[i] += Math.random() * 0.5 + 0.5;
        }
    }
    
    animate() {
        this.draw();
        this.animationId = requestAnimationFrame(() => this.animate());
    }
    
    destroy() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
    }
}

let matrixRain = null;

function initMatrixRain() {
    const canvas = document.getElementById('matrixCanvas');
    const loginPage = document.getElementById('loginPage');
    
    if (canvas && loginPage && !loginPage.classList.contains('hidden')) {
        if (!matrixRain) {
            matrixRain = new MatrixRain(canvas);
        }
    } else if (matrixRain) {
        matrixRain.destroy();
        matrixRain = null;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    authToken = localStorage.getItem('authToken');
    if (authToken) {
        showDashboard();
        loadDashboard();
        console.log('[TOR] Starting aggressive Tor status checks...');
        checkTorStatus();
        
        let rapidCheckCount = 0;
        const rapidCheckInterval = setInterval(() => {
            rapidCheckCount++;
            console.log(`[TOR] Rapid check ${rapidCheckCount}/15...`);
            checkTorStatus();
            if (rapidCheckCount >= 15) {
                clearInterval(rapidCheckInterval);
                console.log('[TOR] Switching to normal check interval...');
                let normalCheckCount = 0;
                const normalCheckInterval = setInterval(() => {
                    normalCheckCount++;
                    checkTorStatus();
                    if (normalCheckCount >= 12) {
                        clearInterval(normalCheckInterval);
                        setInterval(() => {
                            checkTorStatus();
                        }, 15000);
                    }
                }, 5000);
            }
        }, 2000);
    } else {
        showLogin();
        setTimeout(() => initMatrixRain(), 100);
    }

    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
    
    document.getElementById('logoutBtn').addEventListener('click', handleLogout);
    
    document.getElementById('searchInput').addEventListener('input', debounce(handleSearch, 300));
    document.getElementById('categoryFilter').addEventListener('change', () => {
        currentPage = 1;
        loadEntries();
    });
    
    document.getElementById('refreshBtn').addEventListener('click', () => {
        loadDashboard();
        loadEntries();
        checkTorStatus();
    });
    
    document.getElementById('prevPage').addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            loadEntries();
        }
    });
    
    document.getElementById('nextPage').addEventListener('click', () => {
        currentPage++;
        loadEntries();
    });
    
    document.getElementById('closeModal').addEventListener('click', closeModal);
    document.getElementById('cancelChanges').addEventListener('click', closeModal);
    document.getElementById('saveChanges').addEventListener('click', saveEntryChanges);
    
    document.getElementById('modalCriticality').addEventListener('input', (e) => {
        document.getElementById('modalCriticalityValue').textContent = e.target.value;
        updateCriticalityBadge(e.target.value);
    });
    
    document.getElementById('hamburgerBtn').addEventListener('click', toggleSidebar);
    
    document.getElementById('sidebarOverlay').addEventListener('click', closeSidebar);
    
    document.querySelectorAll('.sidebar-item').forEach(btn => {
        btn.addEventListener('click', () => {
            const tabName = btn.getAttribute('data-tab');
            switchTab(tabName);
            if (window.innerWidth < 1025) {
                closeSidebar();
            }
        });
    });
    
    document.getElementById('entriesTabSearchInput').addEventListener('input', debounce(() => {
        entriesTabPage = 1;
        loadEntriesTab();
    }, 300));
    document.getElementById('entriesTabCategoryFilter').addEventListener('change', () => {
        entriesTabPage = 1;
        loadEntriesTab();
    });
    document.getElementById('entriesTabRefreshBtn').addEventListener('click', loadEntriesTab);
    document.getElementById('entriesTabPrevPage').addEventListener('click', () => {
        if (entriesTabPage > 1) {
            entriesTabPage--;
            loadEntriesTab();
        }
    });
    document.getElementById('entriesTabNextPage').addEventListener('click', () => {
        entriesTabPage++;
        loadEntriesTab();
    });
    
    document.getElementById('criticalityRangeFilter').addEventListener('change', () => {
        criticalityTabPage = 1;
        loadCriticalityTab();
    });
    document.getElementById('criticalityRefreshBtn').addEventListener('click', loadCriticalityTab);
    document.getElementById('criticalityPrevPage').addEventListener('click', () => {
        if (criticalityTabPage > 1) {
            criticalityTabPage--;
            loadCriticalityTab();
        }
    });
    document.getElementById('criticalityNextPage').addEventListener('click', () => {
        criticalityTabPage++;
        loadCriticalityTab();
    });
    
    document.getElementById('addSourceBtn').addEventListener('click', () => openSourceModal());
    const triggerScrapeBtn = document.getElementById('triggerScrapeBtn');
    if (triggerScrapeBtn) {
        triggerScrapeBtn.addEventListener('click', triggerManualScrape);
    } else {
        console.error('triggerScrapeBtn not found in DOM');
    }
    document.getElementById('closeSourceModal').addEventListener('click', closeSourceModal);
    document.getElementById('cancelSource').addEventListener('click', closeSourceModal);
    document.getElementById('saveSource').addEventListener('click', saveSource);
    
    document.getElementById('chatToggleBtn').addEventListener('click', toggleChatPanel);
    document.getElementById('chatSendBtn').addEventListener('click', sendChatMessage);
    document.getElementById('chatInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendChatMessage();
        }
    });
    
    document.getElementById('aiAnalysisSearchInput').addEventListener('input', debounce(() => {
        aiAnalysisSearchQuery = document.getElementById('aiAnalysisSearchInput').value;
        aiAnalysisTabPage = 1;
        loadAIAnalysisTab();
    }, 300));
    document.getElementById('aiAnalysisRefreshBtn').addEventListener('click', loadAIAnalysisTab);
    document.getElementById('aiAnalysisPrevPage').addEventListener('click', () => {
        if (aiAnalysisTabPage > 1) {
            aiAnalysisTabPage--;
            loadAIAnalysisTab();
        }
    });
    document.getElementById('aiAnalysisNextPage').addEventListener('click', () => {
        aiAnalysisTabPage++;
        loadAIAnalysisTab();
    });
}

function showLogin() {
    document.getElementById('loginPage').classList.remove('hidden');
    document.getElementById('dashboard').classList.add('hidden');
    const chatPanel = document.getElementById('chatPanel');
    if (chatPanel) {
        chatPanel.style.display = 'none';
    }
    setTimeout(() => initMatrixRain(), 100);
}

function showDashboard() {
    document.getElementById('loginPage').classList.add('hidden');
    document.getElementById('dashboard').classList.remove('hidden');
    const chatPanel = document.getElementById('chatPanel');
    if (chatPanel) {
        chatPanel.style.display = 'flex';
    }
    if (matrixRain) {
        matrixRain.destroy();
        matrixRain = null;
    }
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const errorDiv = document.getElementById('loginError');
    
    errorDiv.textContent = '';
    
    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            errorDiv.textContent = data.error || 'Login failed';
            return;
        }
        
        authToken = data.token;
        localStorage.setItem('authToken', authToken);
        showDashboard();
        loadDashboard();
    } catch (error) {
        errorDiv.textContent = 'Network error. Please try again.';
    }
}

function handleLogout() {
    authToken = null;
    localStorage.removeItem('authToken');
    showLogin();
    document.getElementById('loginForm').reset();
}

function getAuthHeaders() {
    return {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
    };
}

async function loadDashboard() {
    try {
        const response = await fetch(`${API_BASE}/dashboard/stats`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load dashboard');
        }
        
        const stats = await response.json();
        console.log('[DASHBOARD] Stats loaded:', stats);
        updateStats(stats);
        
        const overviewTab = document.getElementById('overviewTab');
        const isOverviewActive = overviewTab && overviewTab.classList.contains('active');
        
        console.log('[DASHBOARD] Overview tab check:', {
            exists: !!overviewTab,
            isActive: isOverviewActive,
            className: overviewTab?.className
        });
        
        setTimeout(() => {
            console.log('[DASHBOARD] Calling updateCharts with stats...');
            updateCharts(stats);
        }, isOverviewActive ? 400 : 600);
        
        loadCategories();
        loadEntries();
        updateAIBadge();
        
        setTimeout(() => {
            const indicator = document.getElementById('torStatusIndicator');
            if (indicator && !indicator.classList.contains('checking')) {
                checkTorStatus();
            }
        }, 500);
    } catch (error) {
        console.error('Error loading dashboard:', error);
    }
}

async function checkTorStatus() {
    const indicator = document.getElementById('torStatusIndicator');
    const icon = document.getElementById('torStatusIcon');
    const text = document.getElementById('torStatusText');
    
    if (indicator && icon && text && !indicator.classList.contains('connected')) {
        indicator.className = 'tor-status-indicator checking';
        icon.textContent = '🔄';
        text.textContent = 'Checking Tor...';
    }
    
    try {
        const response = await fetch(`${API_BASE}/tor/status`, {
            headers: getAuthHeaders(),
            signal: AbortSignal.timeout(8000)
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to check Tor status');
        }
        
        const status = await response.json();
        
        console.log('[TOR] Status response:', {
            is_connected: status.is_connected,
            exit_ip: status.exit_ip,
            message: status.message
        });
        
        if (status.is_connected && status.exit_ip && status.exit_ip.trim() !== '') {
            if (indicator && icon && text) {
                indicator.className = 'tor-status-indicator connected';
                icon.textContent = '🔒';
                const cleanIP = status.exit_ip.trim();
                text.textContent = `Tor: ${cleanIP}`;
                console.log(`[TOR] ✓ Connected! Exit IP: ${cleanIP}`);
            }
        } else if (status.is_connected) {
            if (indicator && icon && text) {
                indicator.className = 'tor-status-indicator checking';
                icon.textContent = '🔄';
                const bootstrapMatch = status.message?.match(/(\d+)%/);
                if (bootstrapMatch) {
                    text.textContent = `Tor: Bootstrapping ${bootstrapMatch[1]}%`;
                } else {
                    text.textContent = 'Tor: Getting IP...';
                    console.warn('[TOR] Connected but IP not available yet. Message:', status.message);
                }
            }
        } else {
            if (indicator && icon && text) {
                indicator.className = 'tor-status-indicator disconnected';
                icon.textContent = '⚠️';
                if (status.message && status.message.includes('Bootstrap')) {
                    const bootstrapMatch = status.message.match(/(\d+)%/);
                    if (bootstrapMatch) {
                        text.textContent = `Tor: Bootstrapping ${bootstrapMatch[1]}%`;
                        indicator.className = 'tor-status-indicator checking';
                    } else {
                        text.textContent = 'Tor: Starting...';
                    }
                } else {
                    text.textContent = 'Tor: Disconnected';
                    console.warn('[TOR] Disconnected. Message:', status.message);
                }
            }
        }
    } catch (error) {
        if (error.name === 'TimeoutError' || error.name === 'AbortError') {
            console.warn('[TOR] Status check timed out');
        } else {
            console.error('[TOR] Error checking Tor status:', error);
        }
        if (indicator && icon && text && !indicator.classList.contains('connected')) {
            indicator.className = 'tor-status-indicator disconnected';
            icon.textContent = '⚠️';
            text.textContent = 'Tor: Checking...';
        }
    }
}

function updateStats(stats) {
    document.getElementById('totalEntries').textContent = stats.total_entries || 0;
    document.getElementById('totalSources').textContent = stats.total_sources || 0;
    
    const highCriticality = stats.criticality_distribution?.find(d => d.range === '81-100')?.count || 0;
    document.getElementById('highCriticality').textContent = highCriticality;
    
    document.getElementById('totalCategories').textContent = stats.category_stats?.length || 0;
    
    document.getElementById('sidebarEntriesBadge').textContent = stats.total_entries || 0;
    document.getElementById('sidebarSourcesBadge').textContent = stats.total_sources || 0;
    document.getElementById('sidebarCriticalityBadge').textContent = highCriticality;
    document.getElementById('sidebarCategoriesBadge').textContent = stats.category_stats?.length || 0;
}

function updateCharts(stats) {
    console.log('[CHARTS] ===== updateCharts STARTED =====');
    console.log('[CHARTS] Stats received:', {
        category_stats_count: stats.category_stats?.length || 0,
        criticality_distribution_count: stats.criticality_distribution?.length || 0,
        category_stats: stats.category_stats,
        criticality_distribution: stats.criticality_distribution
    });
    
    if (typeof Chart === 'undefined') {
        console.error('[CHARTS] Chart.js is not loaded! Waiting for Chart.js...');
        setTimeout(() => {
            console.log('[CHARTS] Retrying after Chart.js load...');
            updateCharts(stats);
        }, 500);
        return;
    }
    
    console.log('[CHARTS] Chart.js is loaded, version:', Chart.version || 'unknown');
    
    const neonColors = [
        '#00d4ff',
        '#7c3aed',
        '#ff3366',
        '#ff6b35',
        '#ffa726',
        '#00ff88',
        '#ff2d55',
        '#5ac8fa',
    ];
    
    const categoryChartElement = document.getElementById('categoryChart');
    if (!categoryChartElement) {
        console.error('[CHARTS] Category chart canvas element NOT FOUND!');
        console.error('[CHARTS] Searching for chart elements...', {
            allCharts: Array.from(document.querySelectorAll('[id*="chart"]')).map(el => ({
                id: el.id,
                tagName: el.tagName,
                visible: window.getComputedStyle(el).display !== 'none'
            })),
            overviewTab: document.getElementById('overviewTab') ? 'found' : 'not found',
                overviewTabActive: document.getElementById('overviewTab')?.classList.contains('active')
            });
        setTimeout(() => {
            console.log('[CHARTS] Retrying category chart render...');
            updateCharts(stats);
        }, 300);
        return;
    }
    
    console.log('[CHARTS] Category chart element FOUND!', {
        id: categoryChartElement.id,
        display: window.getComputedStyle(categoryChartElement).display,
        visibility: window.getComputedStyle(categoryChartElement).visibility,
        width: categoryChartElement.offsetWidth,
        height: categoryChartElement.offsetHeight,
        parentDisplay: window.getComputedStyle(categoryChartElement.parentElement).display
    });
    
    const categoryData = stats.category_stats || [];
    const categoryCard = categoryChartElement.closest('.chart-card');
    
    categoryChartElement.style.display = 'block';
    categoryChartElement.style.visibility = 'visible';
    
    if (window.categoryChart) {
        try {
            window.categoryChart.destroy();
        } catch (e) {
            console.warn('[CHARTS] Error destroying previous category chart:', e);
        }
    }
    
    if (!categoryData || categoryData.length === 0) {
        categoryChartElement.style.display = 'none';
        let placeholder = categoryCard.querySelector('.chart-placeholder');
        if (!placeholder) {
            placeholder = document.createElement('div');
            placeholder.className = 'chart-placeholder';
            categoryCard.appendChild(placeholder);
        }
        placeholder.innerHTML = '<p style="color: var(--text-secondary); padding: 60px 20px; text-align: center; font-size: 14px;">📊 Henüz kategori verisi yok.<br>Kaynak ekleyip tarama yaptıktan sonra burada görünecektir.</p>';
        console.log('[CHARTS] No category data, showing placeholder');
    } else {
        const placeholder = categoryCard.querySelector('.chart-placeholder');
        if (placeholder) {
            placeholder.remove();
        }
        categoryChartElement.style.display = 'block';
        
        console.log('[CHARTS] Rendering category chart with', categoryData.length, 'categories');
        const categoryCtx = categoryChartElement.getContext('2d');
        
        try {
            window.categoryChart = new Chart(categoryCtx, {
            type: 'doughnut',
            data: {
                labels: categoryData.map(c => c.category || 'Uncategorized'),
                datasets: [{
                    data: categoryData.map(c => c.count || 0),
                    backgroundColor: neonColors.slice(0, categoryData.length),
                    borderColor: neonColors.map(c => c + '80'),
                    borderWidth: 2,
                }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                animation: {
                    animateRotate: true,
                    animateScale: true,
                    duration: 1500,
                    easing: 'easeOutQuart',
                },
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: {
                            color: '#e8eaed',
                            font: {
                                size: 12,
                            },
                            padding: 15,
                        },
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#e8eaed',
                        bodyColor: '#e8eaed',
                        borderColor: '#00d4ff',
                        borderWidth: 1,
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed || 0;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                return `${label}: ${value} (%${percentage})`;
                            },
                        },
                    },
                },
            },
        });
        console.log('[CHARTS] Category chart rendered successfully');
        } catch (error) {
            console.error('[CHARTS] Error rendering category chart:', error);
            categoryChartElement.style.display = 'none';
            let placeholder = categoryCard.querySelector('.chart-placeholder');
            if (!placeholder) {
                placeholder = document.createElement('div');
                placeholder.className = 'chart-placeholder';
                categoryCard.appendChild(placeholder);
            }
            placeholder.innerHTML = '<p style="color: var(--text-secondary); padding: 60px 20px; text-align: center; font-size: 14px;">❌ Grafik yüklenirken hata oluştu: ' + error.message + '</p>';
        }
    }
    
    const criticalityChartElement = document.getElementById('criticalityChart');
    if (!criticalityChartElement) {
        console.error('[CHARTS] Criticality chart canvas element NOT FOUND!');
        setTimeout(() => {
            console.log('[CHARTS] Retrying criticality chart render...');
            updateCharts(stats);
        }, 300);
        return;
    }
    
    console.log('[CHARTS] Criticality chart element FOUND!', {
        id: criticalityChartElement.id,
        display: window.getComputedStyle(criticalityChartElement).display,
        visibility: window.getComputedStyle(criticalityChartElement).visibility,
        width: criticalityChartElement.offsetWidth,
        height: criticalityChartElement.offsetHeight,
        parentDisplay: window.getComputedStyle(criticalityChartElement.parentElement).display
    });
    
    const criticalityData = stats.criticality_distribution || [];
    const criticalityCard = criticalityChartElement.closest('.chart-card');
    
    criticalityChartElement.style.display = 'block';
    criticalityChartElement.style.visibility = 'visible';
    criticalityChartElement.style.opacity = '1';
    
    if (window.criticalityChart) {
        try {
            window.criticalityChart.destroy();
        } catch (e) {
            console.warn('[CHARTS] Error destroying previous criticality chart:', e);
        }
    }
    
    if (!criticalityData || criticalityData.length === 0 || criticalityData.every(d => !d.count || d.count === 0)) {
        criticalityChartElement.style.display = 'none';
        let placeholder = criticalityCard.querySelector('.chart-placeholder');
        if (!placeholder) {
            placeholder = document.createElement('div');
            placeholder.className = 'chart-placeholder';
            criticalityCard.appendChild(placeholder);
        }
        placeholder.innerHTML = '<p style="color: var(--text-secondary); padding: 60px 20px; text-align: center; font-size: 14px;">⚠️ Henüz kritiklik verisi yok.<br>Kayıtlar eklendikçe burada görünecektir.</p>';
        console.log('[CHARTS] No criticality data, showing placeholder');
    } else {
        const placeholder = criticalityCard.querySelector('.chart-placeholder');
        if (placeholder) {
            placeholder.remove();
        }
        criticalityChartElement.style.display = 'block';
        
        console.log('[CHARTS] Rendering criticality chart with', criticalityData.length, 'ranges:', criticalityData);
        const criticalityCtx = criticalityChartElement.getContext('2d');
        
        const getBarColor = (range) => {
            if (range.includes('81-100') || range.includes('91-100')) return '#ff3366';
            if (range.includes('61-80') || range.includes('71-80')) return '#ff6b35';
            if (range.includes('41-60') || range.includes('51-60')) return '#ffa726';
            return '#66bb6a';
        };
        
        try {
            window.criticalityChart = new Chart(criticalityCtx, {
            type: 'bar',
            data: {
                labels: criticalityData.map(d => d.range || 'N/A'),
                datasets: [{
                    label: 'Kayıt Sayısı',
                    data: criticalityData.map(d => d.count || 0),
                    backgroundColor: criticalityData.map(d => getBarColor(d.range || '')),
                    borderColor: criticalityData.map(d => getBarColor(d.range || '')),
                    borderWidth: 2,
                    borderRadius: 8,
                }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                animation: {
                    duration: 1500,
                    easing: 'easeOutQuart',
                },
                plugins: {
                    legend: {
                        display: false,
                    },
                    tooltip: {
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#e8eaed',
                        bodyColor: '#e8eaed',
                        borderColor: '#00d4ff',
                        borderWidth: 1,
                        callbacks: {
                            label: function(context) {
                                return `${context.dataset.label}: ${context.parsed.y}`;
                            },
                        },
                    },
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            color: '#9aa0a6',
                            stepSize: 1,
                        },
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)',
                        },
                    },
                    x: {
                        ticks: {
                            color: '#9aa0a6',
                        },
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)',
                        },
                    },
            },
        },
    });
    console.log('[CHARTS] Criticality chart rendered successfully');
    } catch (error) {
        console.error('[CHARTS] Error rendering criticality chart:', error);
        criticalityChartElement.style.display = 'none';
        let placeholder = criticalityCard.querySelector('.chart-placeholder');
        if (!placeholder) {
            placeholder = document.createElement('div');
            placeholder.className = 'chart-placeholder';
            criticalityCard.appendChild(placeholder);
        }
        placeholder.innerHTML = '<p style="color: var(--text-secondary); padding: 60px 20px; text-align: center; font-size: 14px;">❌ Grafik yüklenirken hata oluştu: ' + error.message + '</p>';
    }
    }
    
    console.log('[CHARTS] All charts rendering completed');
    
    const timeSeriesCtx = document.getElementById('timeSeriesChart');
    if (timeSeriesCtx) {
        const timeSeriesData = stats.time_series_data || [];
        
        if (window.timeSeriesChart) {
            window.timeSeriesChart.destroy();
        }
        
        if (timeSeriesData.length === 0) {
            timeSeriesCtx.parentElement.innerHTML = '<p style="color: var(--text-secondary); padding: 40px; text-align: center;">Henüz yeterli veri yok. Paylaşım tarihi olan kayıtlar burada görünecektir.</p>';
        } else {
            window.timeSeriesChart = new Chart(timeSeriesCtx, {
                type: 'line',
                data: {
                    labels: timeSeriesData.map(d => {
                        const date = new Date(d.date);
                        return date.toLocaleDateString('tr-TR', { day: '2-digit', month: 'short' });
                    }),
                    datasets: [{
                        label: 'Toplanan İçerik Sayısı',
                        data: timeSeriesData.map(d => d.count),
                        borderColor: '#00d4ff',
                        backgroundColor: 'rgba(0, 212, 255, 0.1)',
                        borderWidth: 3,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 4,
                        pointBackgroundColor: '#00d4ff',
                        pointBorderColor: '#fff',
                        pointBorderWidth: 2,
                    }],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    animation: {
                        duration: 1500,
                        easing: 'easeOutQuart',
                    },
                    plugins: {
                        legend: {
                            display: true,
                            position: 'top',
                            labels: {
                                color: '#e8eaed',
                                font: {
                                    size: 12,
                                },
                            },
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            titleColor: '#e8eaed',
                            bodyColor: '#e8eaed',
                            borderColor: '#00d4ff',
                            borderWidth: 1,
                        },
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            ticks: {
                                color: '#9aa0a6',
                                stepSize: 1,
                            },
                            grid: {
                                color: 'rgba(255, 255, 255, 0.1)',
                            },
                        },
                        x: {
                            ticks: {
                                color: '#9aa0a6',
                            },
                            grid: {
                                color: 'rgba(255, 255, 255, 0.1)',
                            },
                        },
                    },
                },
            });
        }
    }
    
    // AI Analysis Status Chart
    const aiAnalysisCtx = document.getElementById('aiAnalysisChart');
    if (aiAnalysisCtx && stats.ai_analysis_status) {
        const aiStatus = stats.ai_analysis_status;
        
        if (window.aiAnalysisChart) {
            window.aiAnalysisChart.destroy();
        }
        
        const total = aiStatus.with_analysis + aiStatus.without_analysis;
        
        if (total === 0) {
            aiAnalysisCtx.parentElement.innerHTML = '<p style="color: var(--text-secondary); padding: 40px; text-align: center;">Henüz kayıt yok.</p>';
        } else {
            window.aiAnalysisChart = new Chart(aiAnalysisCtx, {
                type: 'doughnut',
                data: {
                    labels: ['AI Analizi Tamamlandı', 'AI Analizi Bekliyor'],
                    datasets: [{
                        data: [aiStatus.with_analysis, aiStatus.without_analysis],
                        backgroundColor: ['#7c3aed', 'rgba(124, 58, 237, 0.3)'],
                        borderColor: ['#7c3aed', 'rgba(124, 58, 237, 0.5)'],
                        borderWidth: 2,
                    }],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    animation: {
                        animateRotate: true,
                        animateScale: true,
                        duration: 1500,
                        easing: 'easeOutQuart',
                    },
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                color: '#e8eaed',
                                font: {
                                    size: 12,
                                },
                                padding: 15,
                                generateLabels: function(chart) {
                                    const data = chart.data;
                                    if (data.labels.length && data.datasets.length) {
                                        return data.labels.map((label, i) => {
                                            const value = data.datasets[0].data[i];
                                            const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                            return {
                                                text: `${label} (${value} - %${percentage})`,
                                                fillStyle: data.datasets[0].backgroundColor[i],
                                                strokeStyle: data.datasets[0].borderColor[i],
                                                lineWidth: data.datasets[0].borderWidth,
                                                hidden: false,
                                                index: i,
                                            };
                                        });
                                    }
                                    return [];
                                },
                            },
                        },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    const label = context.label || '';
                                    const value = context.parsed || 0;
                                    const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                    return `${label}: ${value} (%${percentage})`;
                                },
                            },
                        },
                    },
                },
            });
        }
    }
}

async function loadCategories() {
    try {
        const response = await fetch(`${API_BASE}/categories`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            throw new Error('Failed to load categories');
        }
        
        const data = await response.json();
        const select = document.getElementById('categoryFilter');
        
        // Keep "All Categories" option
        select.innerHTML = '<option value="">All Categories</option>';
        
        data.categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category;
            option.textContent = category;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading categories:', error);
    }
}

async function loadEntries() {
    const search = document.getElementById('searchInput').value;
    const category = document.getElementById('categoryFilter').value;
    
    const params = new URLSearchParams({
        page: currentPage,
        pageSize: pageSize,
    });
    
    if (category) {
        params.append('category', category);
    }
    
    if (search) {
        params.append('search', search);
    }
    
    try {
        const response = await fetch(`${API_BASE}/entries?${params}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load entries');
        }
        
        const data = await response.json();
        displayEntries(data.entries);
        updatePagination(data.total, data.page, data.pageSize);
    } catch (error) {
        console.error('Error loading entries:', error);
        document.getElementById('entriesTableBody').innerHTML = 
            '<tr><td colspan="6" class="loading">Error loading entries</td></tr>';
    }
}

// Critical keywords to highlight
const CRITICAL_KEYWORDS = ['exploit', 'breach', 'leak', 'ransomware', 'malware', 'attack', 'vulnerability', 'zero-day', 'phishing', 'ddos', 'hack', 'compromise', 'exfiltrat', 'trojan', 'backdoor', 'apt'];
const HIGH_PRIORITY_KEYWORDS = ['alert', 'warning', 'critical', 'urgent', 'threat', 'risk', 'exposed', 'stolen', 'data breach'];

function highlightKeywords(text) {
    let highlighted = escapeHtml(text);
    
    // Highlight critical keywords
    CRITICAL_KEYWORDS.forEach(keyword => {
        const regex = new RegExp(`\\b(${keyword})\\b`, 'gi');
        highlighted = highlighted.replace(regex, '<span class="critical-keyword">$1</span>');
    });
    
    // Highlight high priority keywords
    HIGH_PRIORITY_KEYWORDS.forEach(keyword => {
        const regex = new RegExp(`\\b(${keyword})\\b`, 'gi');
        highlighted = highlighted.replace(regex, '<span class="highlight-keyword">$1</span>');
    });
    
    return highlighted;
}

function getCriticalityLevel(score) {
    if (score >= 80) return 'high';
    if (score >= 50) return 'medium';
    return 'low';
}

function displayEntries(entries) {
    const tbody = document.getElementById('entriesTableBody');
    
    if (entries.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No entries found</td></tr>';
        return;
    }
    
    tbody.innerHTML = entries.map(entry => {
        const criticalityLevel = getCriticalityLevel(entry.criticality_score);
        const highlightedTitle = highlightKeywords(entry.title);
        const hasAIAnalysis = entry.ai_analysis && entry.ai_analysis.trim() !== '';
        const aiBadge = hasAIAnalysis 
            ? '<span class="ai-badge" title="AI Analysis Available">🤖</span>' 
            : '<span class="ai-badge no-ai" title="AI Analysis Pending">⏳</span>';
        
        return `
        <tr onclick="openEntryModal(${entry.id})" data-criticality="${criticalityLevel}">
            <td>
                <strong class="entry-title">${highlightedTitle}</strong>
            </td>
            <td>${escapeHtml(entry.source_name)}</td>
            <td>
                <span class="category-badge">${escapeHtml(entry.category)}</span>
            </td>
            <td>
                <span class="criticality-badge ${getCriticalityClass(entry.criticality_score)}">
                    ${entry.criticality_score}
                </span>
            </td>
            <td>
                ${entry.share_date ? `<strong>${formatDate(entry.share_date)}</strong><br><small style="color: var(--text-secondary); font-size: 0.85em;">Eklenme: ${formatDate(entry.created_at)}</small>` : `<span style="color: var(--text-secondary);">Paylaşım: Bilinmiyor</span><br><small style="color: var(--text-secondary); font-size: 0.85em;">Eklenme: ${formatDate(entry.created_at)}</small>`}
            </td>
            <td style="display: flex; align-items: center; gap: 8px;">
                ${aiBadge}
                <button class="action-btn" onclick="event.stopPropagation(); openEntryModal(${entry.id})">
                    View
                </button>
            </td>
        </tr>
    `;
    }).join('');
}

function updatePagination(total, page, size) {
    const start = (page - 1) * size + 1;
    const end = Math.min(page * size, total);
    
    document.getElementById('paginationInfo').textContent = `Showing ${start}-${end} of ${total}`;
    document.getElementById('pageInfo').textContent = `Page ${page}`;
    
    document.getElementById('prevPage').disabled = page <= 1;
    document.getElementById('nextPage').disabled = end >= total;
}

async function openEntryModal(entryId) {
    currentEntryId = entryId;
    
    try {
        const response = await fetch(`${API_BASE}/entries/${entryId}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            throw new Error('Failed to load entry');
        }
        
        const entry = await response.json();
        populateModal(entry);
        document.getElementById('entryModal').classList.remove('hidden');
    } catch (error) {
        console.error('Error loading entry:', error);
        alert('Failed to load entry details');
    }
}

function populateModal(entry) {
    document.getElementById('modalTitle').textContent = entry.title;
    document.getElementById('modalSource').textContent = `${entry.source_name} (${entry.source_url})`;
    document.getElementById('modalContent').textContent = entry.cleaned_content;
    // CRITICAL: Share Date vs Created At distinction
    // Share Date = when content was published on source
    // Created At = when we added it to our system
    // CRITICAL: Share Date vs Created At distinction
    // Share Date = when content was published on source
    // Created At = when we added it to our system
    const shareDateEl = document.getElementById('modalShareDate');
    const shareDateLabel = shareDateEl.parentElement.querySelector('label');
    if (entry.share_date) {
        shareDateEl.textContent = formatDate(entry.share_date);
        shareDateEl.style.color = '';
        shareDateEl.style.fontStyle = '';
        shareDateLabel.textContent = 'Paylaşım Tarihi:';
    } else {
        shareDateEl.textContent = 'Bilinmiyor';
        shareDateEl.style.color = 'var(--text-secondary)';
        shareDateEl.style.fontStyle = 'italic';
        shareDateLabel.textContent = 'Paylaşım Tarihi:';
    }
    
    const createdAtEl = document.getElementById('modalCreatedAt');
    const createdAtLabel = createdAtEl.parentElement.querySelector('label');
    createdAtEl.textContent = formatDate(entry.created_at);
    createdAtLabel.textContent = 'Sisteme Eklenme Tarihi:';
    
    // Set criticality
    const criticalitySlider = document.getElementById('modalCriticality');
    criticalitySlider.value = entry.criticality_score;
    document.getElementById('modalCriticalityValue').textContent = entry.criticality_score;
    updateCriticalityBadge(entry.criticality_score);
    
    // Display AI analysis if available (secondary, supporting information)
    const aiAnalysisGroup = document.getElementById('aiAnalysisGroup');
    const aiAnalysisContent = document.getElementById('modalAIAnalysis');
    if (entry.ai_analysis && entry.ai_analysis.trim() !== '') {
        aiAnalysisContent.innerHTML = highlightKeywords(entry.ai_analysis);
        aiAnalysisGroup.style.display = 'block';
    } else {
        // Show placeholder if AI analysis is pending
        aiAnalysisContent.innerHTML = '<em style="color: var(--text-secondary); opacity: 0.7;">AI analizi henüz hazır değil. Analiz tamamlandığında burada görünecektir.</em>';
        aiAnalysisGroup.style.display = 'block';
    }
    
    // Set category
    const categorySelect = document.getElementById('modalCategory');
    categorySelect.innerHTML = '';
    
    // Load categories for select
    fetch(`${API_BASE}/categories`, {
        headers: getAuthHeaders(),
    })
    .then(res => res.json())
    .then(data => {
        data.categories.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat;
            option.textContent = cat;
            if (cat === entry.category) {
                option.selected = true;
            }
            categorySelect.appendChild(option);
        });
        
        // Add current category if not in list
        if (!data.categories.includes(entry.category)) {
            const option = document.createElement('option');
            option.value = entry.category;
            option.textContent = entry.category;
            option.selected = true;
            categorySelect.appendChild(option);
        }
    });
}

function updateCriticalityBadge(score) {
    const valueEl = document.getElementById('modalCriticalityValue');
    valueEl.className = `criticality-badge ${getCriticalityClass(score)}`;
}

function closeModal() {
    document.getElementById('entryModal').classList.add('hidden');
    currentEntryId = null;
}

async function saveEntryChanges() {
    if (!currentEntryId) return;
    
    const criticality = parseInt(document.getElementById('modalCriticality').value);
    const category = document.getElementById('modalCategory').value;
    
    try {
        // Update criticality
        await fetch(`${API_BASE}/entries/${currentEntryId}/criticality`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({ score: criticality }),
        });
        
        // Update category
        await fetch(`${API_BASE}/entries/${currentEntryId}/category`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({ category: category }),
        });
        
        closeModal();
        loadDashboard();
        loadEntries();
    } catch (error) {
        console.error('Error saving changes:', error);
        alert('Failed to save changes');
    }
}

function handleSearch() {
    currentPage = 1;
    loadEntries();
}

function getCriticalityClass(score) {
    if (score >= 80) return 'criticality-high';
    if (score >= 50) return 'criticality-medium';
    return 'criticality-low';
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Sidebar Management
function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    const overlay = document.getElementById('sidebarOverlay');
    const hamburger = document.getElementById('hamburgerBtn');
    
    sidebar.classList.toggle('open');
    overlay.classList.toggle('active');
    hamburger.classList.toggle('active');
}

function closeSidebar() {
    const sidebar = document.getElementById('sidebar');
    const overlay = document.getElementById('sidebarOverlay');
    const hamburger = document.getElementById('hamburgerBtn');
    
    sidebar.classList.remove('open');
    overlay.classList.remove('active');
    hamburger.classList.remove('active');
}

function openSidebar() {
    const sidebar = document.getElementById('sidebar');
    const overlay = document.getElementById('sidebarOverlay');
    const hamburger = document.getElementById('hamburgerBtn');
    
    sidebar.classList.add('open');
    overlay.classList.add('active');
    hamburger.classList.add('active');
}

// Tab Management
function switchTab(tabName) {
    currentTab = tabName;
    
    // Update sidebar buttons
    document.querySelectorAll('.sidebar-item').forEach(btn => {
        btn.classList.remove('active');
        if (btn.getAttribute('data-tab') === tabName) {
            btn.classList.add('active');
        }
    });
    
    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    const activeContent = document.getElementById(`${tabName}Tab`);
    if (activeContent) {
        activeContent.classList.add('active');
    }
    
    // Load tab-specific data
    switch(tabName) {
        case 'overview':
            // Reload dashboard to ensure charts are rendered when tab becomes visible
            console.log('[TABS] Switched to overview tab, reloading dashboard...');
            setTimeout(() => {
                loadDashboard();
            }, 150);
            break;
        case 'entries':
            loadEntriesTab();
            break;
        case 'sources':
            loadSourcesTab();
            break;
        case 'criticality':
            loadCriticalityTab();
            break;
        case 'categories':
            loadCategoriesTab();
            break;
        case 'ai-analysis':
            loadAIAnalysisTab();
            break;
    }
}

// Load Entries Tab
async function loadEntriesTab() {
    const search = document.getElementById('entriesTabSearchInput').value;
    const category = document.getElementById('entriesTabCategoryFilter').value;
    
    // Load categories for filter dropdown
    try {
        const categoriesResponse = await fetch(`${API_BASE}/categories`, {
            headers: getAuthHeaders(),
        });
        if (categoriesResponse.ok) {
            const categoriesData = await categoriesResponse.json();
            const select = document.getElementById('entriesTabCategoryFilter');
            select.innerHTML = '<option value="">All Categories</option>';
            categoriesData.categories.forEach(cat => {
                const option = document.createElement('option');
                option.value = cat;
                option.textContent = cat;
                if (cat === category) {
                    option.selected = true;
                }
                select.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Error loading categories:', error);
    }
    
    const params = new URLSearchParams({
        page: entriesTabPage,
        pageSize: pageSize,
    });
    
    if (category) {
        params.append('category', category);
    }
    
    if (search) {
        params.append('search', search);
    }
    
    try {
        const response = await fetch(`${API_BASE}/entries?${params}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load entries');
        }
        
        const data = await response.json();
        displayEntriesTab(data.entries);
        updateEntriesTabPagination(data.total, data.page, data.pageSize);
    } catch (error) {
        console.error('Error loading entries:', error);
        document.getElementById('entriesTabTableBody').innerHTML = 
            '<tr><td colspan="6" class="loading">Error loading entries</td></tr>';
    }
}

function displayEntriesTab(entries) {
    const tbody = document.getElementById('entriesTabTableBody');
    
    if (entries.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No entries found</td></tr>';
        return;
    }
    
    tbody.innerHTML = entries.map(entry => `
        <tr onclick="openEntryModal(${entry.id})">
            <td>
                <strong>${escapeHtml(entry.title)}</strong>
            </td>
            <td>${escapeHtml(entry.source_name)}</td>
            <td>
                <span class="category-badge">${escapeHtml(entry.category)}</span>
            </td>
            <td>
                <span class="criticality-badge ${getCriticalityClass(entry.criticality_score)}">
                    ${entry.criticality_score}
                </span>
            </td>
            <td>${formatDate(entry.created_at)}</td>
            <td>
                <button class="action-btn" onclick="event.stopPropagation(); openEntryModal(${entry.id})">
                    View
                </button>
            </td>
        </tr>
    `).join('');
}

function updateEntriesTabPagination(total, page, size) {
    const start = (page - 1) * size + 1;
    const end = Math.min(page * size, total);
    
    document.getElementById('entriesTabPaginationInfo').textContent = `Showing ${start}-${end} of ${total}`;
    document.getElementById('entriesTabPageInfo').textContent = `Page ${page}`;
    
    document.getElementById('entriesTabPrevPage').disabled = page <= 1;
    document.getElementById('entriesTabNextPage').disabled = end >= total;
}

// Load Sources Tab
async function loadSourcesTab() {
    try {
        const response = await fetch(`${API_BASE}/dashboard/stats`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load sources');
        }
        
        const stats = await response.json();
        displaySources(stats);
    } catch (error) {
        console.error('Error loading sources:', error);
        document.getElementById('sourcesGrid').innerHTML = '<div class="loading">Error loading sources</div>';
    }
}

async function displaySources(stats) {
    try {
        const response = await fetch(`${API_BASE}/sources`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load sources');
        }
        
        const data = await response.json();
        const sources = data.sources || [];
        
        const grid = document.getElementById('sourcesGrid');
        document.getElementById('sourcesCount').textContent = `${sources.length} active sources`;
        
        if (sources.length === 0) {
            grid.innerHTML = '<div class="loading">No sources found. Click "Add Source" to add a new source.</div>';
            return;
        }
        
        // Fetch scrape status for each source
        let scrapeStatuses = {};
        try {
            const statusResponse = await fetch(`${API_BASE}/scraper/status`, {
                headers: getAuthHeaders(),
            });
            if (statusResponse.ok) {
                const statusData = await statusResponse.json();
                statusData.active_scrapes.forEach(state => {
                    scrapeStatuses[state.source_id] = state;
                });
            }
        } catch (error) {
            console.error('Error fetching scrape status:', error);
        }
        
        grid.innerHTML = sources.map(source => {
            const scrapeState = scrapeStatuses[source.id];
            let statusBadge = '';
            let buttonState = '';
            
            if (scrapeState) {
                if (scrapeState.status === 'running') {
                    statusBadge = '<span class="scrape-status-badge running">⏳ Scraping...</span>';
                    buttonState = 'disabled';
                } else if (scrapeState.status === 'completed') {
                    statusBadge = `<span class="scrape-status-badge completed">✅ Completed (${scrapeState.entries_inserted} entries)</span>`;
                } else if (scrapeState.status === 'failed') {
                    statusBadge = `<span class="scrape-status-badge failed">❌ Failed: ${escapeHtml(scrapeState.error)}</span>`;
                }
            }
            
            return `
            <div class="source-card" data-source-id="${source.id}">
                <div class="source-card-header">
                    <div class="source-name">${escapeHtml(source.name)}</div>
                    ${statusBadge}
                </div>
                <div class="source-url">${escapeHtml(source.url)}</div>
                <div class="source-stats">
                    <div class="source-stat">
                        <span>📅</span>
                        <span>Added: ${formatDate(source.created_at)}</span>
                    </div>
                </div>
                <div class="source-actions" style="margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap;">
                    <button class="category-btn scrape-btn" onclick="scrapeSource(${source.id})" ${buttonState} style="background: var(--primary-color); color: white; border: none;">
                        🔄 Tarama
                    </button>
                    <button class="category-btn" onclick="editSource(${source.id}, '${escapeHtml(source.name)}', '${escapeHtml(source.url)}')">
                        Edit
                    </button>
                    <button class="category-btn" onclick="deleteSource(${source.id})" style="background: var(--danger); color: white; border: none;">
                        Delete
                    </button>
                </div>
            </div>
        `;
        }).join('');
    } catch (error) {
        console.error('Error loading sources:', error);
        document.getElementById('sourcesGrid').innerHTML = '<div class="loading">Error loading sources</div>';
    }
}

// Load Criticality Tab
async function loadCriticalityTab() {
    const range = document.getElementById('criticalityRangeFilter').value;
    const [min, max] = range.split('-').map(Number);
    
    const params = new URLSearchParams({
        page: criticalityTabPage,
        pageSize: pageSize,
    });
    
    try {
        const response = await fetch(`${API_BASE}/entries?${params}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load critical entries');
        }
        
        const data = await response.json();
        const filteredEntries = data.entries.filter(entry => 
            entry.criticality_score >= min && entry.criticality_score <= max
        );
        
        displayCriticalityTab(filteredEntries);
        updateCriticalityTabPagination(filteredEntries.length, criticalityTabPage, pageSize);
    } catch (error) {
        console.error('Error loading critical entries:', error);
        document.getElementById('criticalityTableBody').innerHTML = 
            '<tr><td colspan="6" class="loading">Error loading critical entries</td></tr>';
    }
}

function displayCriticalityTab(entries) {
    const tbody = document.getElementById('criticalityTableBody');
    
    if (entries.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No entries found in this criticality range</td></tr>';
        return;
    }
    
    tbody.innerHTML = entries.map(entry => `
        <tr onclick="openEntryModal(${entry.id})">
            <td>
                <strong>${escapeHtml(entry.title)}</strong>
            </td>
            <td>${escapeHtml(entry.source_name)}</td>
            <td>
                <span class="category-badge">${escapeHtml(entry.category)}</span>
            </td>
            <td>
                <span class="criticality-badge ${getCriticalityClass(entry.criticality_score)}">
                    ${entry.criticality_score}
                </span>
            </td>
            <td>${formatDate(entry.created_at)}</td>
            <td>
                <button class="action-btn" onclick="event.stopPropagation(); openEntryModal(${entry.id})">
                    View
                </button>
            </td>
        </tr>
    `).join('');
}

function updateCriticalityTabPagination(total, page, size) {
    const start = (page - 1) * size + 1;
    const end = Math.min(page * size, total);
    
    document.getElementById('criticalityPaginationInfo').textContent = `Showing ${start}-${end} of ${total}`;
    document.getElementById('criticalityPageInfo').textContent = `Page ${page}`;
    
    document.getElementById('criticalityPrevPage').disabled = page <= 1;
    document.getElementById('criticalityNextPage').disabled = end >= total;
}

// Load Categories Tab
async function loadCategoriesTab() {
    const grid = document.getElementById('categoriesGrid');
    if (!grid) {
        console.error('categoriesGrid element not found');
        return;
    }
    
    grid.innerHTML = '<div class="loading">Loading categories...</div>';
    
    try {
        const statsResponse = await fetch(`${API_BASE}/dashboard/stats`, {
            headers: getAuthHeaders(),
        });
        
        if (!statsResponse.ok) {
            if (statsResponse.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load category stats');
        }
        
        const stats = await statsResponse.json();
        console.log('Category stats received:', stats);
        
        // Check both possible field names
        const categoryStats = stats.category_stats || stats.categoryStats || [];
        console.log('Category stats array:', categoryStats);
        
        if (categoryStats.length === 0) {
            grid.innerHTML = '<div class="loading">No categories found. Data may still be loading.</div>';
            return;
        }
        
        displayCategories(categoryStats);
    } catch (error) {
        console.error('Error loading categories:', error);
        grid.innerHTML = '<div class="loading">Error loading categories: ' + error.message + '</div>';
    }
}

function displayCategories(categoryStats) {
    const grid = document.getElementById('categoriesGrid');
    
    if (!grid) {
        console.error('categoriesGrid element not found in displayCategories');
        return;
    }
    
    if (!categoryStats || categoryStats.length === 0) {
        grid.innerHTML = '<div class="loading">No categories found. Please wait for data to be collected.</div>';
        return;
    }
    
    const categoryDescriptions = {
        'Malware Analysis': 'Analysis of malicious software behavior, infection vectors, and data exfiltration methods.',
        'Data Breach': 'Information regarding unauthorized access to sensitive data and potential impact assessment.',
        'Vulnerability Disclosure': 'Technical details about security vulnerabilities and exploitation vectors.',
        'Threat Intelligence': 'Intelligence reports on emerging threats and recommended defensive measures.',
        'Security Research': 'Research findings on security-related topics and defensive strategies.',
        'Cyber Attack': 'Reports on active or recent cyber attacks, including attack vectors and mitigation strategies.',
        'Exploit Development': 'Technical analysis of exploit development techniques and proof-of-concept implementations.',
        'Network Security': 'Analysis of network security issues, intrusion patterns, and defensive configurations.',
        'Uncategorized': 'Entries that have not been assigned to a specific category yet.',
    };
    
    try {
        grid.innerHTML = categoryStats.map(cat => {
            const categoryName = cat.category || cat.Category || 'Unknown';
            const categoryCount = cat.count || cat.Count || 0;
            const description = categoryDescriptions[categoryName] || 'Security-related content requiring analysis and categorization.';
            
            return `
                <div class="category-card" onclick="filterByCategory('${escapeHtml(categoryName)}')">
                    <div class="category-card-header">
                        <div class="category-name">${escapeHtml(categoryName)}</div>
                        <div class="category-count">${categoryCount}</div>
                    </div>
                    <div class="category-description">
                        ${description}
                    </div>
                    <div class="category-actions">
                        <button class="category-btn" onclick="event.stopPropagation(); filterByCategory('${escapeHtml(categoryName)}')">
                            View Entries
                        </button>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Error displaying categories:', error);
        grid.innerHTML = '<div class="loading">Error displaying categories: ' + error.message + '</div>';
    }
}

// AI Analysis Tab variables
let aiAnalysisTabPage = 1;
let aiAnalysisSearchQuery = '';

// Load AI Analysis Tab
async function loadAIAnalysisTab() {
    const params = new URLSearchParams({
        page: aiAnalysisTabPage,
        pageSize: pageSize,
    });
    
    if (aiAnalysisSearchQuery) {
        params.append('search', aiAnalysisSearchQuery);
    }
    
    try {
        const response = await fetch(`${API_BASE}/entries?${params}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Failed to load AI analyzed entries');
        }
        
        const data = await response.json();
        // Filter entries that have AI analysis
        const aiAnalyzedEntries = data.entries.filter(entry => 
            entry.ai_analysis && entry.ai_analysis.trim() !== ''
        );
        
        displayAIAnalysisTab(aiAnalyzedEntries, data.total || aiAnalyzedEntries.length);
        
        // Update sidebar badge
        document.getElementById('sidebarAIBadge').textContent = aiAnalyzedEntries.length;
    } catch (error) {
        console.error('Error loading AI analyzed entries:', error);
        document.getElementById('aiAnalysisTableBody').innerHTML = 
            '<tr><td colspan="7" class="loading">Error loading AI analyzed entries</td></tr>';
    }
}

function displayAIAnalysisTab(entries, total) {
    const tbody = document.getElementById('aiAnalysisTableBody');
    
    if (!tbody) {
        console.error('aiAnalysisTableBody element not found');
        return;
    }
    
    if (entries.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" class="loading">
                    No entries with AI analysis found. AI analysis will appear here once entries are analyzed.
                </td>
            </tr>
        `;
        document.getElementById('aiAnalysisPaginationInfo').textContent = 'Showing 0-0 of 0';
        document.getElementById('aiAnalysisPageInfo').textContent = 'Page 1';
        document.getElementById('aiAnalysisPrevPage').disabled = true;
        document.getElementById('aiAnalysisNextPage').disabled = true;
        return;
    }
    
    const start = (aiAnalysisTabPage - 1) * pageSize + 1;
    const end = Math.min(start + entries.length - 1, total);
    
    tbody.innerHTML = entries.map(entry => {
        const aiPreview = entry.ai_analysis 
            ? (entry.ai_analysis.length > 100 
                ? entry.ai_analysis.substring(0, 100) + '...' 
                : entry.ai_analysis)
            : 'No analysis';
        
        return `
            <tr data-criticality="${getCriticalityLevel(entry.criticality_score)}">
                <td class="entry-title">${highlightKeywords(escapeHtml(entry.title))}</td>
                <td>${escapeHtml(entry.source_name || 'Unknown')}</td>
                <td><span class="category-badge">${escapeHtml(entry.category || 'Uncategorized')}</span></td>
                <td>
                    <span class="criticality-badge ${getCriticalityClass(entry.criticality_score)}">
                        ${entry.criticality_score}
                    </span>
                </td>
                <td class="ai-preview">${highlightKeywords(escapeHtml(aiPreview))}</td>
                <td>${formatDate(entry.created_at)}</td>
                <td>
                    <button class="action-btn" onclick="viewEntry(${entry.id})">View</button>
                </td>
            </tr>
        `;
    }).join('');
    
    document.getElementById('aiAnalysisPaginationInfo').textContent = `Showing ${start}-${end} of ${total}`;
    document.getElementById('aiAnalysisPageInfo').textContent = `Page ${aiAnalysisTabPage}`;
    document.getElementById('aiAnalysisPrevPage').disabled = aiAnalysisTabPage <= 1;
    document.getElementById('aiAnalysisNextPage').disabled = end >= total;
}

// Chat Panel Functions
function toggleChatPanel() {
    const panel = document.getElementById('chatPanel');
    const icon = document.getElementById('chatToggleIcon');
    panel.classList.toggle('collapsed');
    icon.textContent = panel.classList.contains('collapsed') ? '+' : '−';
}

async function sendChatMessage() {
    const input = document.getElementById('chatInput');
    const sendBtn = document.getElementById('chatSendBtn');
    const messagesContainer = document.getElementById('chatMessages');
    const message = input.value.trim();
    
    if (!message) {
        return;
    }
    
    // Disable input and button
    input.disabled = true;
    sendBtn.disabled = true;
    
    // Add user message
    addChatMessage(message, 'user');
    input.value = '';
    
    // Update status
    document.getElementById('chatStatus').textContent = 'Thinking...';
    
    // Try streaming first, fallback to regular if not supported
    const useStreaming = true; // Enable streaming for better UX
    
    if (useStreaming) {
        try {
            await sendChatMessageStreaming(message);
        } catch (error) {
            console.error('Streaming failed, falling back to regular:', error);
            // Fallback to regular request
            await sendChatMessageRegular(message);
        }
    } else {
        await sendChatMessageRegular(message);
    }
    
    // Re-enable input and button
    input.disabled = false;
    sendBtn.disabled = false;
    input.focus();
    document.getElementById('chatStatus').textContent = 'Ready';
}

// Streaming chat using SSE
async function sendChatMessageStreaming(message) {
    const headers = getAuthHeaders();
    headers['Accept'] = 'text/event-stream';
    
    const response = await fetch(`${API_BASE}/chat`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify({ message, stream: true }),
    });
    
    if (!response.ok) {
        if (response.status === 401) {
            handleLogout();
            return;
        }
        throw new Error('Chat service unavailable');
    }
    
    // Create AI message container for streaming
    const aiMessageDiv = document.createElement('div');
    aiMessageDiv.className = 'chat-message ai-message';
    aiMessageDiv.style.opacity = '0';
    aiMessageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-avatar">🤖</div>
            <div class="message-text" id="streamingMessage"></div>
        </div>
    `;
    
    const messagesContainer = document.getElementById('chatMessages');
    messagesContainer.appendChild(aiMessageDiv);
    
    // Trigger fade-in
    setTimeout(() => {
        aiMessageDiv.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
        aiMessageDiv.style.opacity = '1';
        aiMessageDiv.style.transform = 'translateY(0)';
    }, 10);
    
    const streamingText = document.getElementById('streamingMessage');
    let fullText = '';
    
    // Read SSE stream
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    
    try {
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            
            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');
            
            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.slice(6);
                    if (data === 'done' || data === '[DONE]') {
                        break;
                    }
                    if (data.startsWith('{')) {
                        try {
                            const jsonData = JSON.parse(data);
                            if (jsonData.error) {
                                throw new Error(jsonData.error);
                            }
                        } catch (e) {
                            // Not JSON, treat as text
                        }
                    }
                    if (data && data !== 'done' && !data.startsWith('{')) {
                        fullText += data;
                        streamingText.innerHTML = highlightKeywords(escapeHtml(fullText));
                        
                        // Auto-scroll
                        messagesContainer.scrollTo({
                            top: messagesContainer.scrollHeight,
                            behavior: 'smooth'
                        });
                    }
                }
            }
        }
    } catch (error) {
        console.error('Stream read error:', error);
        streamingText.innerHTML = highlightKeywords(escapeHtml('Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.'));
    }
}

// Regular (non-streaming) chat
async function sendChatMessageRegular(message) {
    try {
        const response = await fetch(`${API_BASE}/chat`, {
            method: 'POST',
            headers: getAuthHeaders(),
            body: JSON.stringify({ message, stream: false }),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Chat service unavailable');
        }
        
        const data = await response.json();
        
        // Add AI response (always show reply, even if it's a fallback message)
        if (data.reply) {
            addChatMessage(data.reply, 'ai');
        } else {
            // Fallback if no reply field
            addChatMessage('Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.', 'ai');
        }
    } catch (error) {
        // Network errors, etc. - show soft fallback message
        console.error('Chat error:', error);
        addChatMessage('Lokal AI chat servisi şu anda kullanılamıyor. Lütfen daha sonra tekrar deneyin.', 'ai');
    }
}

function addChatMessage(text, role) {
    const messagesContainer = document.getElementById('chatMessages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${role}-message`;
    
    const avatar = role === 'user' ? '👤' : '🤖';
    const highlightedText = highlightKeywords(escapeHtml(text));
    
    // Add fade-in animation
    messageDiv.style.opacity = '0';
    messageDiv.style.transform = 'translateY(10px)';
    
    messageDiv.innerHTML = `
        <div class="message-content">
            <div class="message-avatar">${avatar}</div>
            <div class="message-text">${highlightedText}</div>
        </div>
    `;
    
    messagesContainer.appendChild(messageDiv);
    
    // Trigger fade-in animation
    setTimeout(() => {
        messageDiv.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
        messageDiv.style.opacity = '1';
        messageDiv.style.transform = 'translateY(0)';
    }, 10);
    
    // Scroll to bottom smoothly
    setTimeout(() => {
        messagesContainer.scrollTo({
            top: messagesContainer.scrollHeight,
            behavior: 'smooth'
        });
    }, 50);
}

function filterByCategory(category) {
    switchTab('entries');
    document.getElementById('entriesTabCategoryFilter').value = category;
    entriesTabPage = 1;
    loadEntriesTab();
}

// Update AI badge count
async function updateAIBadge() {
    try {
        const params = new URLSearchParams({
            page: 1,
            pageSize: 1000, // Get enough entries to count
        });
        
        const response = await fetch(`${API_BASE}/entries?${params}`, {
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            return; // Fail silently
        }
        
        const data = await response.json();
        const aiAnalyzedCount = data.entries.filter(entry => 
            entry.ai_analysis && entry.ai_analysis.trim() !== ''
        ).length;
        
        const badgeElement = document.getElementById('sidebarAIBadge');
        if (badgeElement) {
            badgeElement.textContent = aiAnalyzedCount;
        }
    } catch (error) {
        // Fail silently - badge update is not critical
        console.error('Error updating AI badge:', error);
    }
}

// Source Management Functions
let currentSourceId = null;

function openSourceModal(sourceId = null, name = '', url = '') {
    currentSourceId = sourceId;
    const modal = document.getElementById('sourceModal');
    const title = document.getElementById('sourceModalTitle');
    const nameInput = document.getElementById('sourceName');
    const urlInput = document.getElementById('sourceURL');
    const errorDiv = document.getElementById('sourceError');
    
    if (sourceId) {
        title.textContent = 'Edit Source';
        nameInput.value = name;
        urlInput.value = url;
    } else {
        title.textContent = 'Add Source';
        nameInput.value = '';
        urlInput.value = '';
    }
    
    errorDiv.textContent = '';
    modal.classList.remove('hidden');
}

function closeSourceModal() {
    document.getElementById('sourceModal').classList.add('hidden');
    currentSourceId = null;
    document.getElementById('sourceForm').reset();
    document.getElementById('sourceError').textContent = '';
}

async function saveSource() {
    const name = document.getElementById('sourceName').value.trim();
    const url = document.getElementById('sourceURL').value.trim();
    const errorDiv = document.getElementById('sourceError');
    
    errorDiv.textContent = '';
    
    if (!name || !url) {
        errorDiv.textContent = 'Please fill in all fields';
        return;
    }
    
    // NO URL validation - accept whatever user enters as-is
    // Backend will handle validation if needed
    
    try {
        const apiUrl = currentSourceId 
            ? `${API_BASE}/sources/${currentSourceId}`
            : `${API_BASE}/sources`;
        
        const method = currentSourceId ? 'PUT' : 'POST';
        
        const response = await fetch(apiUrl, {
            method: method,
            headers: getAuthHeaders(),
            body: JSON.stringify({ name, url }),
        });
        
        const data = await response.json();
        
        if (!response.ok) {
            errorDiv.textContent = data.error || 'Failed to save source';
            return;
        }
        
        closeSourceModal();
        loadSourcesTab();
        loadDashboard(); // Refresh stats
    } catch (error) {
        errorDiv.textContent = 'Network error: ' + error.message;
    }
}

function editSource(id, name, url) {
    openSourceModal(id, name, url);
}

async function deleteSource(id) {
    if (!confirm('Are you sure you want to delete this source? This will also delete all entries from this source.')) {
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/sources/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            const data = await response.json();
            alert('Failed to delete source: ' + (data.error || 'Unknown error'));
            return;
        }
        
        loadSourcesTab();
        loadDashboard(); // Refresh stats
    } catch (error) {
        alert('Network error: ' + error.message);
    }
}

// Trigger scrape for a specific source
async function scrapeSource(sourceId) {
    console.log('[FRONTEND] Scrape button clicked for source ID:', sourceId);
    
    const sourceCards = document.querySelectorAll('.source-card');
    let scrapeBtn = null;
    
    // Find the button for this source
    for (const card of sourceCards) {
        const actions = card.querySelector('.source-actions');
        if (actions) {
            const btn = actions.querySelector(`button[onclick*="scrapeSource(${sourceId})"]`);
            if (btn) {
                scrapeBtn = btn;
                break;
            }
        }
    }
    
    const originalText = scrapeBtn ? scrapeBtn.textContent : '🔄 Tarama';
    
    // Update button state
    if (scrapeBtn) {
        scrapeBtn.disabled = true;
        scrapeBtn.textContent = '⏳ Tarama başlatılıyor...';
    }
    
    try {
        console.log(`[FRONTEND] Sending POST request to: ${API_BASE}/sources/${sourceId}/scrape`);
        
        const response = await fetch(`${API_BASE}/sources/${sourceId}/scrape`, {
            method: 'POST',
            headers: getAuthHeaders(),
        });
        
        console.log(`[FRONTEND] Response status: ${response.status}`);
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            const errorData = await response.json().catch(() => ({}));
            throw new Error(errorData.message || errorData.error || 'Tarama başlatılamadı');
        }
        
        const data = await response.json();
        console.log('[FRONTEND] Response data:', data);
        
        // Show success message
        if (scrapeBtn) {
            scrapeBtn.textContent = '⏳ Scraping...';
            scrapeBtn.disabled = true;
        }
        
        // Show notification
        showNotification('Tarama başlatıldı! Arka planda çalışıyor...', 'success');
        
        // Start polling for scrape status
        startScrapeStatusPolling(sourceId);
        
        // Refresh dashboard after scrape completes
        pollForScrapeCompletion(sourceId, () => {
            loadDashboard();
            loadSourcesTab();
            if (scrapeBtn) {
                scrapeBtn.disabled = false;
                scrapeBtn.textContent = originalText;
            }
        });
        
    } catch (error) {
        console.error('[FRONTEND] Error triggering source scrape:', error);
        if (scrapeBtn) {
            scrapeBtn.textContent = '❌ Hata';
            scrapeBtn.disabled = false;
            
            // Show error notification
            showNotification('Tarama başlatılamadı: ' + error.message, 'error');
            
            // Reset button after 3 seconds
            setTimeout(() => {
                scrapeBtn.textContent = originalText;
            }, 3000);
        }
    }
}

// Trigger manual scrape of all sources
async function triggerManualScrape() {
    const btn = document.getElementById('triggerScrapeBtn');
    const originalText = btn.textContent;
    
    // Disable button and show loading state
    btn.disabled = true;
    btn.textContent = '⏳ Tarama başlatılıyor...';
    
    try {
        const response = await fetch(`${API_BASE}/scraper/trigger`, {
            method: 'POST',
            headers: getAuthHeaders(),
        });
        
        if (!response.ok) {
            if (response.status === 401) {
                handleLogout();
                return;
            }
            throw new Error('Tarama başlatılamadı');
        }
        
        const data = await response.json();
        
        // Show success message
        btn.textContent = '⏳ Scraping...';
        btn.disabled = true;
        
        // Show notification
        showNotification('Tüm kaynaklar taranıyor...', 'success');
        
        // Refresh dashboard periodically while scraping
        const refreshInterval = setInterval(() => {
            loadDashboard();
            loadSourcesTab();
        }, 5000);
        
        // Stop refreshing after 2 minutes
        setTimeout(() => {
            clearInterval(refreshInterval);
            btn.disabled = false;
            btn.textContent = originalText;
            loadDashboard();
            loadSourcesTab();
        }, 120000);
        
    } catch (error) {
        console.error('Error triggering scrape:', error);
        btn.textContent = '❌ Hata: ' + error.message;
        btn.disabled = false;
        
        // Reset button after 3 seconds
        setTimeout(() => {
            btn.textContent = originalText;
        }, 3000);
    }
}

// Poll for scrape status
let scrapePollIntervals = {};

function startScrapeStatusPolling(sourceId) {
    // Clear existing interval if any
    if (scrapePollIntervals[sourceId]) {
        clearInterval(scrapePollIntervals[sourceId]);
    }
    
    // Poll every 2 seconds
    scrapePollIntervals[sourceId] = setInterval(async () => {
        try {
            const response = await fetch(`${API_BASE}/scraper/status/${sourceId}`, {
                headers: getAuthHeaders(),
            });
            
            if (response.ok) {
                const state = await response.json();
                updateSourceScrapeStatus(sourceId, state);
                
                // Stop polling if scrape is completed or failed
                if (state.status === 'completed' || state.status === 'failed') {
                    clearInterval(scrapePollIntervals[sourceId]);
                    delete scrapePollIntervals[sourceId];
                }
            }
        } catch (error) {
            console.error('Error polling scrape status:', error);
        }
    }, 2000);
}

function updateSourceScrapeStatus(sourceId, state) {
    const sourceCard = document.querySelector(`.source-card[data-source-id="${sourceId}"]`);
    if (!sourceCard) return;
    
    const header = sourceCard.querySelector('.source-card-header');
    if (!header) return;
    
    // Remove existing status badge
    const existingBadge = header.querySelector('.scrape-status-badge');
    if (existingBadge) {
        existingBadge.remove();
    }
    
    // Add new status badge
    let badge = '';
    if (state.status === 'running') {
        badge = '<span class="scrape-status-badge running">⏳ Scraping...</span>';
    } else if (state.status === 'completed') {
        badge = `<span class="scrape-status-badge completed">✅ Completed (${state.entries_inserted} entries)</span>`;
    } else if (state.status === 'failed') {
        badge = `<span class="scrape-status-badge failed">❌ Failed: ${escapeHtml(state.error)}</span>`;
    }
    
    if (badge) {
        header.insertAdjacentHTML('beforeend', badge);
    }
    
    // Update button state
    const scrapeBtn = sourceCard.querySelector('.scrape-btn');
    if (scrapeBtn) {
        if (state.status === 'running') {
            scrapeBtn.disabled = true;
            scrapeBtn.textContent = '⏳ Scraping...';
        } else {
            scrapeBtn.disabled = false;
            scrapeBtn.textContent = '🔄 Tarama';
        }
    }
}

function pollForScrapeCompletion(sourceId, onComplete) {
    const maxAttempts = 60; // 2 minutes max
    let attempts = 0;
    
    const checkInterval = setInterval(async () => {
        attempts++;
        
        try {
            const response = await fetch(`${API_BASE}/scraper/status/${sourceId}`, {
                headers: getAuthHeaders(),
            });
            
            if (response.ok) {
                const state = await response.json();
                if (state.status === 'completed' || state.status === 'failed') {
                    clearInterval(checkInterval);
                    if (onComplete) onComplete();
                }
            }
        } catch (error) {
            console.error('Error checking scrape completion:', error);
        }
        
        if (attempts >= maxAttempts) {
            clearInterval(checkInterval);
            console.warn('Scrape completion check timed out');
        }
    }, 2000);
}

