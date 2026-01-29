<script>
    import { onMount, onDestroy } from "svelte";
    import { flip } from "svelte/animate";
    import { quintOut } from "svelte/easing";
    // IMPORTANT: Ensure these paths match your Wails project structure
    // usually located in frontend/src/wailsjs/go/main/App.js
    import { TrackTicker, ToggleDemoMode } from "../wailsjs/go/main/App";
    import { EventsOn } from "../wailsjs/runtime/runtime";

    // --- STATE ---
    let stocks = [];
    let gainers = [];
    let losers = [];

    // UI State
    let searchInput = "";
    let showSettings = false;
    let isDemoMode = false;
    let stopListener;

    // --- LIFECYCLE ---
    onMount(() => {
        // Listen for the event emitted by Go
        stopListener = EventsOn("market_update", (data) => {
            stocks = data || []; // Safety check
            updateLists();
        });
    });

    onDestroy(() => {
        if (stopListener) stopListener();
    });

    // --- LOGIC ---

    function updateLists() {
        // 1. Split into Gainers (Green) and Losers (Red)
        let g = stocks.filter((s) => s.change >= 0);
        let l = stocks.filter((s) => s.change < 0);

        // 2. Sorting Function
        // Priority: VIPs always on top, then sorted by magnitude of change
        const sortFn = (descending) => (a, b) => {
            // VIP Priority
            if (a.is_vip && !b.is_vip) return -1; // A moves up
            if (!a.is_vip && b.is_vip) return 1; // B moves up

            // Secondary Sort: Change %
            // If descending (Gainers): Highest % first
            // If ascending (Losers): Lowest (most negative) % first
            return descending ? b.change - a.change : a.change - b.change;
        };

        g.sort(sortFn(true));
        l.sort(sortFn(false));

        gainers = g;
        losers = l;
    }

    async function handleSearch(e) {
        if (e.key === "Enter" && searchInput.trim() !== "") {
            // Call Go Backend
            await TrackTicker(searchInput);
            searchInput = ""; // Clear input
        }
    }

    function toggleDemo() {
        isDemoMode = !isDemoMode;
        // Notify Go Backend
        ToggleDemoMode(isDemoMode);
    }
</script>

<main
    class="h-screen w-screen bg-slate-950 text-slate-100 font-sans flex flex-col relative overflow-hidden"
>
    <header
        class="h-16 bg-slate-900 border-b border-slate-800 flex items-center justify-between px-6 shrink-0 z-20 shadow-lg relative"
    >
        <div class="flex items-center gap-4 w-48">
            <h1
                class="text-xl font-bold tracking-widest text-white select-none"
            >
                MARKET<span class="font-light text-slate-400">WATCH</span>
            </h1>
        </div>

        <div class="flex-1 max-w-lg mx-4">
            <div class="relative group">
                <div
                    class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
                >
                    <svg
                        class="h-4 w-4 text-slate-500 group-focus-within:text-emerald-400 transition-colors"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                        />
                    </svg>
                </div>
                <input
                    type="text"
                    bind:value={searchInput}
                    on:keydown={handleSearch}
                    placeholder="Add Ticker (e.g. NVDA, GME)..."
                    class="block w-full pl-10 pr-3 py-2 border border-slate-700 rounded-md leading-5 bg-slate-800 text-slate-300 placeholder-slate-500 focus:outline-none focus:bg-slate-900 focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500 sm:text-sm transition-all shadow-inner"
                />
            </div>
        </div>

        <div class="flex items-center gap-4 w-48 justify-end">
            <button
                on:click={() => (showSettings = !showSettings)}
                class="text-slate-400 hover:text-white transition-colors p-2 hover:bg-slate-800 rounded-full"
                title="Settings"
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                    />
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                </svg>
            </button>

            <div
                class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider px-3 py-1 rounded-full border {isDemoMode
                    ? 'bg-amber-900/20 border-amber-500/30 text-amber-400'
                    : 'bg-emerald-900/20 border-emerald-500/30 text-emerald-400'}"
            >
                <span class="relative flex h-2 w-2">
                    <span
                        class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 {isDemoMode
                            ? 'bg-amber-400'
                            : 'bg-emerald-400'}"
                    ></span>
                    <span
                        class="relative inline-flex rounded-full h-2 w-2 {isDemoMode
                            ? 'bg-amber-500'
                            : 'bg-emerald-500'}"
                    ></span>
                </span>
                {isDemoMode ? "DEMO" : "LIVE"}
            </div>
        </div>
    </header>

    {#if showSettings}
        <div
            class="absolute top-16 right-6 w-72 bg-slate-900/95 backdrop-blur border border-slate-700 shadow-2xl rounded-lg p-5 z-50 transition-all origin-top-right"
        >
            <h3
                class="text-xs font-bold text-slate-500 mb-4 uppercase tracking-widest border-b border-slate-800 pb-2"
            >
                Configuration
            </h3>

            <div class="flex items-center justify-between mb-2">
                <div>
                    <span class="text-sm text-slate-200 block font-medium"
                        >Simulation Mode</span
                    >
                    <span class="text-xs text-slate-500 block"
                        >Generate fake market moves.</span
                    >
                </div>
                <button
                    on:click={toggleDemo}
                    class="w-11 h-6 rounded-full relative transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-slate-900 focus:ring-emerald-500 {isDemoMode
                        ? 'bg-emerald-500'
                        : 'bg-slate-700'}"
                >
                    <span
                        class="absolute top-1 left-1 bg-white w-4 h-4 rounded-full transition-transform shadow {isDemoMode
                            ? 'translate-x-5'
                            : ''}"
                    ></span>
                </button>
            </div>
        </div>
        <div
            class="absolute inset-0 z-40"
            on:click={() => (showSettings = false)}
        ></div>
    {/if}

    <div
        class="grid grid-cols-2 h-full overflow-hidden divide-x divide-slate-800"
    >
        <div class="flex flex-col h-full overflow-hidden bg-slate-950/50">
            <div
                class="py-3 px-4 flex items-center justify-between border-b border-slate-800/50 bg-slate-900/30"
            >
                <h2 class="text-xs font-bold tracking-[0.2em] text-emerald-500">
                    GAINERS
                </h2>
                <span
                    class="text-xs font-mono text-slate-500 bg-slate-900 px-2 py-0.5 rounded border border-slate-800"
                    >{gainers.length}</span
                >
            </div>

            <div
                class="flex-1 overflow-y-auto p-3 space-y-2 scroll-smooth custom-scrollbar"
            >
                {#each gainers as stock (stock.symbol)}
                    <div
                        animate:flip={{ duration: 400, easing: quintOut }}
                        class="relative bg-slate-900 p-3 rounded-md border-l-4 border-emerald-500 shadow-lg shadow-black/20 hover:bg-slate-800 transition-colors group"
                    >
                        {#if stock.is_vip}
                            <div
                                class="absolute -top-1 -right-1 z-10"
                                title="High Priority Feed"
                            >
                                <span class="flex h-3 w-3">
                                    <span
                                        class="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"
                                    ></span>
                                    <span
                                        class="relative inline-flex rounded-full h-3 w-3 bg-indigo-500 border-2 border-slate-900"
                                    ></span>
                                </span>
                            </div>
                        {/if}

                        <div class="flex justify-between items-baseline mb-1">
                            <div class="flex items-center gap-2">
                                <span class="font-bold text-sm text-slate-100"
                                    >{stock.symbol}</span
                                >
                                {#if stock.is_vip}
                                    <span
                                        class="text-[9px] font-bold bg-indigo-500/20 text-indigo-300 px-1.5 py-0.5 rounded tracking-wide"
                                        >VIP</span
                                    >
                                {/if}
                            </div>
                            <span
                                class="font-mono font-medium text-sm text-emerald-400"
                                >+{stock.change.toFixed(2)}%</span
                            >
                        </div>
                        <div class="flex justify-between items-center">
                            <span
                                class="text-xs text-slate-400 tabular-nums font-mono"
                                >${stock.price.toFixed(2)}</span
                            >
                            {#if stock.is_live}
                                <span
                                    class="text-[10px] animate-pulse text-yellow-400"
                                    title="Real-time Trade">⚡</span
                                >
                            {/if}
                        </div>
                    </div>
                {/each}
            </div>
        </div>

        <div class="flex flex-col h-full overflow-hidden bg-slate-950/50">
            <div
                class="py-3 px-4 flex items-center justify-between border-b border-slate-800/50 bg-slate-900/30"
            >
                <h2 class="text-xs font-bold tracking-[0.2em] text-rose-500">
                    LOSERS
                </h2>
                <span
                    class="text-xs font-mono text-slate-500 bg-slate-900 px-2 py-0.5 rounded border border-slate-800"
                    >{losers.length}</span
                >
            </div>

            <div
                class="flex-1 overflow-y-auto p-3 space-y-2 scroll-smooth custom-scrollbar"
            >
                {#each losers as stock (stock.symbol)}
                    <div
                        animate:flip={{ duration: 400, easing: quintOut }}
                        class="relative bg-slate-900 p-3 rounded-md border-l-4 border-rose-500 shadow-lg shadow-black/20 hover:bg-slate-800 transition-colors group"
                    >
                        {#if stock.is_vip}
                            <div
                                class="absolute -top-1 -right-1 z-10"
                                title="High Priority Feed"
                            >
                                <span class="flex h-3 w-3">
                                    <span
                                        class="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"
                                    ></span>
                                    <span
                                        class="relative inline-flex rounded-full h-3 w-3 bg-indigo-500 border-2 border-slate-900"
                                    ></span>
                                </span>
                            </div>
                        {/if}

                        <div class="flex justify-between items-baseline mb-1">
                            <div class="flex items-center gap-2">
                                <span class="font-bold text-sm text-slate-100"
                                    >{stock.symbol}</span
                                >
                                {#if stock.is_vip}
                                    <span
                                        class="text-[9px] font-bold bg-indigo-500/20 text-indigo-300 px-1.5 py-0.5 rounded tracking-wide"
                                        >VIP</span
                                    >
                                {/if}
                            </div>
                            <span
                                class="font-mono font-medium text-sm text-rose-400"
                                >{stock.change.toFixed(2)}%</span
                            >
                        </div>
                        <div class="flex justify-between items-center">
                            <span
                                class="text-xs text-slate-400 tabular-nums font-mono"
                                >${stock.price.toFixed(2)}</span
                            >
                            {#if stock.is_live}
                                <span
                                    class="text-[10px] animate-pulse text-yellow-400"
                                    title="Real-time Trade">⚡</span
                                >
                            {/if}
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    </div>
</main>

<style>
    /* Dark scrollbar styling for Webkit browsers (Chrome/Electron/Wails) */
    .custom-scrollbar::-webkit-scrollbar {
        width: 6px;
    }
    .custom-scrollbar::-webkit-scrollbar-track {
        background: #0f172a; /* Slate-900 */
    }
    .custom-scrollbar::-webkit-scrollbar-thumb {
        background: #334155; /* Slate-700 */
        border-radius: 4px;
    }
    .custom-scrollbar::-webkit-scrollbar-thumb:hover {
        background: #475569; /* Slate-600 */
    }
</style>
