class NavDrawer extends HTMLElement {
    connectedCallback() {
        this.innerHTML = `
            <!-- Hamburger Menu Toggle Button -->
            <button id="drawer-toggle" onclick="toggleDrawer(true)" 
                style="position: fixed; top: 1rem; left: 1rem; z-index: 9999; cursor: pointer;"
                class="bg-gray-900 text-gray-300 hover:text-cyan-400 p-2.5 rounded-lg border border-gray-800 shadow-xl transition-colors">
                <svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                </svg>
            </button>

            <!-- Dark Overlay Backdrop -->
            <div id="drawer-backdrop" onclick="toggleDrawer(false)" 
                style="position: fixed; inset: 0; background-color: rgba(0, 0, 0, 0.6); backdrop-filter: blur(4px); z-index: 9998; opacity: 0; pointer-events: none; transition: opacity 0.3s ease;"></div>

            <!-- Slide-Out Drawer Panel -->
            <aside id="drawer-panel" 
                style="position: fixed; top: 0; left: 0; bottom: 0; width: 16rem; z-index: 9999; transform: translateX(-100%); transition: transform 0.3s ease-in-out; display: flex; flex-direction: column;"
                class="bg-gray-900 border-r border-gray-800 shadow-2xl">
                
                <!-- Panel Header -->
                <div class="p-4 border-b border-gray-800 flex justify-between items-center">
                    <span class="font-mono font-bold text-sm text-cyan-400">⚡ NAVIGATION</span>
                    <button onclick="toggleDrawer(false)" class="text-gray-500 hover:text-gray-300 text-xl leading-none" style="cursor: pointer;">&times;</button>
                </div>

                <!-- Link Items -->
                <nav class="flex-1 p-4 space-y-1 font-mono text-xs">
                    <a href="/" class="nav-link flex items-center gap-3 px-3 py-2.5 rounded-md text-gray-400 hover:bg-gray-800 hover:text-gray-100 transition-colors">
                        <span>🏠</span> Home
                    </a>
                    <a href="/modbus/rtu" class="nav-link flex items-center gap-3 px-3 py-2.5 rounded-md text-gray-400 hover:bg-gray-800 hover:text-gray-100 transition-colors">
                        <span>🔌</span> Modbus RTU
                    </a>
                    <a href="/htop" class="nav-link flex items-center gap-3 px-3 py-2.5 rounded-md text-gray-400 hover:bg-gray-800 hover:text-gray-100 transition-colors">
                        <span>📊</span> System Monitor
                    </a>
                    <a href="/tailscale" class="nav-link flex items-center gap-3 px-3 py-2.5 rounded-md text-gray-400 hover:bg-gray-800 hover:text-gray-100 transition-colors">
                        <span>🔒</span> Tailscale
                    </a>
                    <a href="/logs" class="nav-link flex items-center gap-3 px-3 py-2.5 rounded-md text-gray-400 hover:bg-gray-800 hover:text-gray-100 transition-colors">
                        <span>💻</span> Process Logs
                    </a>
                </nav>

                <!-- Panel Footer -->
                <div class="p-4 border-t border-gray-800 text-[11px] font-mono text-gray-600">
                    System Control Panel
                </div>
            </aside>
        `;

        this.highlightActiveLink();
    }

    highlightActiveLink() {
        const currentPath = window.location.pathname;
        const links = this.querySelectorAll('.nav-link');
        links.forEach(link => {
            if (link.getAttribute('href') === currentPath) {
                link.classList.add('bg-cyan-950/50', 'text-cyan-400', 'border', 'border-cyan-800/50', 'font-semibold');
                link.classList.remove('text-gray-400');
            }
        });
    }
}

// Register Web Component
customElements.define('nav-drawer', NavDrawer);

// Open / Close Control Function
function toggleDrawer(open) {
    const panel = document.getElementById('drawer-panel');
    const backdrop = document.getElementById('drawer-backdrop');

    if (!panel || !backdrop) return;

    if (open) {
        panel.style.transform = 'translateX(0)';
        backdrop.style.opacity = '1';
        backdrop.style.pointerEvents = 'auto';
    } else {
        panel.style.transform = 'translateX(-100%)';
        backdrop.style.opacity = '0';
        backdrop.style.pointerEvents = 'none';
    }
}