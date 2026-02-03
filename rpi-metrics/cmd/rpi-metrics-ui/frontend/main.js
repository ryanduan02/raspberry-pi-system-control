// Configuration
const CONFIG = {
    apiEndpoint: '/api/metrics',
    refreshInterval: 2000, // 2 seconds
    maxTemp: 85, // Maximum temperature for progress bar
};

// State
let state = {
    isPaused: false,
    intervalId: null,
    lastData: null,
};

// DOM Elements
const elements = {
    connectionStatus: document.getElementById('connection-status'),
    lastUpdate: document.getElementById('last-update'),
    cpuTempValue: document.getElementById('cpu-temp-value'),
    cpuTempBar: document.getElementById('cpu-temp-bar'),
    cpuTempCard: document.getElementById('cpu-temp-card'),
    cpuUtilValue: document.getElementById('cpu-util-value'),
    cpuUtilBar: document.getElementById('cpu-util-bar'),
    cpuCores: document.getElementById('cpu-cores'),
    coolingValue: document.getElementById('cooling-value'),
    coolingStatus: document.getElementById('cooling-status'),
    storageMounts: document.getElementById('storage-mounts'),
    pauseBtn: document.getElementById('pause-btn'),
    refreshInterval: document.getElementById('refresh-interval'),
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    elements.refreshInterval.textContent = CONFIG.refreshInterval / 1000;
    elements.pauseBtn.addEventListener('click', togglePause);
    startPolling();
});

// Start polling for metrics
function startPolling() {
    fetchMetrics(); // Fetch immediately
    state.intervalId = setInterval(fetchMetrics, CONFIG.refreshInterval);
}

// Toggle pause/resume
function togglePause() {
    state.isPaused = !state.isPaused;
    
    if (state.isPaused) {
        clearInterval(state.intervalId);
        elements.pauseBtn.textContent = 'Resume';
        elements.pauseBtn.classList.add('paused');
    } else {
        startPolling();
        elements.pauseBtn.textContent = 'Pause';
        elements.pauseBtn.classList.remove('paused');
    }
}

// Fetch metrics from API
async function fetchMetrics() {
    try {
        const response = await fetch(CONFIG.apiEndpoint);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        state.lastData = data;
        
        updateUI(data);
        setConnectionStatus(true);
        
    } catch (error) {
        console.error('Failed to fetch metrics:', error);
        setConnectionStatus(false);
    }
}

// Update UI with new data
function updateUI(data) {
    updateLastUpdateTime(data.timestamp);
    
    if (data.metrics) {
        updateCPUTemp(data.metrics.cpu_temp);
        updateCPUUtilization(data.metrics.cpu_utilization);
        updateCooling(data.metrics.cpu_cooling_device);
        updateStorage(data.metrics.storage_usage);
    }
}

// Update connection status indicator
function setConnectionStatus(isOnline) {
    elements.connectionStatus.className = `status-indicator ${isOnline ? 'online' : 'offline'}`;
}

// Update last update timestamp
function updateLastUpdateTime(timestamp) {
    const date = new Date(timestamp);
    const timeStr = date.toLocaleTimeString();
    elements.lastUpdate.textContent = `Last update: ${timeStr}`;
}

// Update CPU Temperature
function updateCPUTemp(samples) {
    if (!samples || samples.length === 0) {
        elements.cpuTempValue.textContent = '--';
        return;
    }
    
    const tempSample = samples.find(s => s.name === 'cpu_temperature');
    if (!tempSample) return;
    
    const temp = tempSample.value;
    const tempRounded = temp.toFixed(1);
    
    elements.cpuTempValue.textContent = tempRounded;
    
    // Update progress bar
    const percentage = Math.min((temp / CONFIG.maxTemp) * 100, 100);
    elements.cpuTempBar.style.width = `${percentage}%`;
    
    // Update temperature color class
    elements.cpuTempValue.className = '';
    if (temp < 50) {
        elements.cpuTempValue.classList.add('temp-cool');
    } else if (temp < 70) {
        elements.cpuTempValue.classList.add('temp-warm');
    } else {
        elements.cpuTempValue.classList.add('temp-hot');
    }
}

// Update CPU Utilization
function updateCPUUtilization(samples) {
    if (!samples || samples.length === 0) {
        elements.cpuUtilValue.textContent = '--';
        elements.cpuCores.innerHTML = '';
        return;
    }
    
    // Find total CPU utilization
    const totalSample = samples.find(s => s.labels && s.labels.cpu === 'total');
    if (totalSample) {
        const util = totalSample.value;
        elements.cpuUtilValue.textContent = util.toFixed(1);
        elements.cpuUtilBar.style.width = `${Math.min(util, 100)}%`;
    }
    
    // Update individual cores
    const coreSamples = samples.filter(s => s.labels && s.labels.cpu && s.labels.cpu.startsWith('cpu'));
    
    if (coreSamples.length > 0) {
        elements.cpuCores.innerHTML = coreSamples.map(sample => {
            const cpuName = sample.labels.cpu;
            const value = sample.value.toFixed(1);
            return `
                <div class="core-item">
                    <div class="core-name">${cpuName.toUpperCase()}</div>
                    <div class="core-value">${value}%</div>
                    <div class="mini-bar">
                        <div class="mini-fill" style="width: ${Math.min(sample.value, 100)}%"></div>
                    </div>
                </div>
            `;
        }).join('');
    }
}

// Update Cooling State
function updateCooling(samples) {
    if (!samples || samples.length === 0) {
        elements.coolingValue.textContent = '--';
        return;
    }
    
    const coolingSample = samples.find(s => s.name === 'cooling_state');
    if (!coolingSample) return;
    
    const state = coolingSample.value;
    elements.coolingValue.textContent = state;
    
    // Update cooling level indicators
    const levels = document.querySelectorAll('.cooling-level');
    levels.forEach(level => {
        const levelNum = parseInt(level.dataset.level);
        if (levelNum <= state) {
            level.classList.add('active');
        } else {
            level.classList.remove('active');
        }
    });
    
    // Update status text
    const statusTexts = ['Fan: Off', 'Fan: Low', 'Fan: Medium', 'Fan: High', 'Fan: Maximum'];
    elements.coolingStatus.textContent = statusTexts[Math.min(state, 4)];
}

// Update Storage
function updateStorage(samples) {
    if (!samples || samples.length === 0) {
        elements.storageMounts.innerHTML = '<p class="error-message">No storage data available</p>';
        return;
    }
    
    // Group samples by mount point
    const mounts = {};
    
    samples.forEach(sample => {
        const mountPoint = sample.labels?.mount_point || sample.labels?.path || 'unknown';
        
        if (!mounts[mountPoint]) {
            mounts[mountPoint] = {};
        }
        
        mounts[mountPoint][sample.name] = sample;
    });
    
    // Generate HTML for each mount
    const mountsHTML = Object.entries(mounts).map(([mountPoint, data]) => {
        const usedPercent = data.storage_used_percent?.value || 0;
        const totalBytes = data.storage_total_bytes?.value || 0;
        const usedBytes = data.storage_used_bytes?.value || 0;
        const availBytes = data.storage_avail_bytes?.value || 0;
        
        const usedStr = formatBytes(usedBytes);
        const totalStr = formatBytes(totalBytes);
        const availStr = formatBytes(availBytes);
        
        // Determine color based on usage
        let colorClass = '';
        if (usedPercent > 90) {
            colorClass = 'temp-hot';
        } else if (usedPercent > 70) {
            colorClass = 'temp-warm';
        }
        
        return `
            <div class="mount-item">
                <div class="mount-header">
                    <span class="mount-path">${mountPoint}</span>
                    <span class="mount-percent ${colorClass}">${usedPercent.toFixed(1)}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill storage" style="width: ${Math.min(usedPercent, 100)}%"></div>
                </div>
                <div class="mount-details">
                    <span>Used: ${usedStr}</span>
                    <span>Free: ${availStr}</span>
                    <span>Total: ${totalStr}</span>
                </div>
            </div>
        `;
    }).join('');
    
    elements.storageMounts.innerHTML = mountsHTML;
}

// Format bytes to human readable
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
