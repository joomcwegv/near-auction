// === CONFIG ===
const CONTRACT_ID = "final-auction2026.testnet";
const NETWORK_ID = "testnet";
const NODE_URL = "https://test.rpc.fastnear.com";
const WALLET_URL = "https://testnet.mynearwallet.com";

// === STATE ===
let near, wallet, account, currentUser, auctionInfo;

// === INIT ===
async function initNear() {
    const nearConfig = {
        networkId: NETWORK_ID,
        keyStore: new nearApi.keyStores.BrowserLocalStorageKeyStore(),
        nodeUrl: NODE_URL,
        walletUrl: WALLET_URL,
    };

    near = await nearApi.connect(nearConfig);
    wallet = new nearApi.WalletConnection(near, "near-auction");

    if (wallet.isSignedIn()) {
        currentUser = wallet.getAccountId();
        account = wallet.account();
        document.getElementById("connectText").textContent = truncateAddress(currentUser);
        document.getElementById("connectBtn").classList.add("connected");
        document.getElementById("placeBidBtn").disabled = false;
    }

    // Load auction data
    await loadAuctionInfo();
    startTimer();

    // Auto-refresh every 10 seconds
    setInterval(loadAuctionInfo, 10000);
}

// === LOAD AUCTION INFO ===
async function loadAuctionInfo() {
    try {
        const provider = new nearApi.providers.JsonRpcProvider(NODE_URL);
        const rawResult = await provider.query({
            request_type: "call_function",
            account_id: CONTRACT_ID,
            method_name: "get_info",
            args_base64: btoa("{}"),
            finality: "final",
        });

        const result = JSON.parse(new TextDecoder().decode(new Uint8Array(rawResult.result)));
        auctionInfo = typeof result === "string" ? JSON.parse(result) : result;

        updateUI();
    } catch (err) {
        console.error("Failed to load auction info:", err);
    }
}

// === UPDATE UI ===
function updateUI() {
    if (!auctionInfo) return;

    // Auctioneer
    document.getElementById("auctioneerDisplay").textContent = truncateAddress(auctionInfo.auctioneer);

    // Highest bid
    const bidNEAR = yoctoToNear(auctionInfo.highest_bid);
    document.getElementById("highestBidDisplay").textContent = `${bidNEAR} NEAR`;

    // Highest bidder
    const bidder = auctionInfo.highest_bidder || "Nobody yet";
    document.getElementById("highestBidderDisplay").textContent = 
        bidder === "Nobody yet" ? bidder : truncateAddress(bidder);

    // Status
    const badge = document.getElementById("statusBadge");
    if (auctionInfo.active) {
        badge.className = "status-badge active";
        badge.innerHTML = '<span class="pulse"></span> Active';
    } else {
        badge.className = "status-badge ended";
        badge.innerHTML = "Ended";
    }

    // Bids list
    const bidsList = document.getElementById("bidsList");
    const bids = auctionInfo.bids || {};
    const entries = Object.entries(bids);

    if (entries.length === 0) {
        bidsList.innerHTML = '<p class="empty-bids">No bids yet. Be the first!</p>';
    } else {
        entries.sort((a, b) => {
            const aVal = BigInt(a[1]);
            const bVal = BigInt(b[1]);
            return bVal > aVal ? 1 : bVal < aVal ? -1 : 0;
        });

        bidsList.innerHTML = entries.map(([bidderAddr, amount], i) => {
            const isWinner = bidderAddr === auctionInfo.highest_bidder;
            return `
                <div class="bid-entry ${isWinner ? 'winner' : ''}">
                    <span class="bidder">${isWinner ? '<span class="crown">👑</span>' : ''}${truncateAddress(bidderAddr)}</span>
                    <span class="bid-amount">${yoctoToNear(amount)} NEAR</span>
                </div>
            `;
        }).join("");
    }

    // Admin panel
    if (currentUser && currentUser === auctionInfo.auctioneer) {
        document.getElementById("admin-section").style.display = "block";
    }
}

// === TIMER ===
let timerInterval;
function startTimer() {
    updateTimer();
    timerInterval = setInterval(updateTimer, 1000);
}

function updateTimer() {
    if (!auctionInfo) return;
    const now = Date.now();
    const end = auctionInfo.end_time;
    const diff = end - now;

    const timerEl = document.getElementById("timerDisplay");

    if (diff <= 0) {
        timerEl.textContent = "ENDED";
        timerEl.style.color = "var(--danger)";
        return;
    }

    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);

    timerEl.textContent = `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
}

// === WALLET ===
document.getElementById("connectBtn").addEventListener("click", () => {
    if (wallet && wallet.isSignedIn()) {
        wallet.signOut();
        location.reload();
    } else if (wallet) {
        wallet.requestSignIn({ contractId: CONTRACT_ID });
    }
});

// === QUICK AMOUNTS ===
document.querySelectorAll(".quick-btn").forEach(btn => {
    btn.addEventListener("click", () => {
        document.getElementById("bidAmount").value = btn.dataset.amount;
    });
});

// === PLACE BID ===
document.getElementById("placeBidBtn").addEventListener("click", async () => {
    const amountStr = document.getElementById("bidAmount").value;
    const statusEl = document.getElementById("bidStatus");

    if (!amountStr || parseFloat(amountStr) <= 0) {
        statusEl.textContent = "Please enter a valid bid amount";
        statusEl.className = "bid-status error";
        return;
    }

    try {
        statusEl.textContent = "⏳ Placing bid...";
        statusEl.className = "bid-status loading";

        const deposit = nearApi.utils.format.parseNearAmount(amountStr);
        
        await account.functionCall({
            contractId: CONTRACT_ID,
            methodName: "place_bid",
            args: {},
            gas: "30000000000000",
            attachedDeposit: deposit,
        });

        statusEl.textContent = "✅ Bid placed successfully!";
        statusEl.className = "bid-status success";
        await loadAuctionInfo();
    } catch (err) {
        console.error("Bid error:", err);
        statusEl.textContent = "❌ " + (err.message || "Failed to place bid");
        statusEl.className = "bid-status error";
    }
});

// === ADMIN ACTIONS ===
document.getElementById("cancelBtn")?.addEventListener("click", async () => {
    try {
        await account.functionCall({
            contractId: CONTRACT_ID, methodName: "cancel",
            args: {}, gas: "100000000000000",
        });
        alert("Auction cancelled! All bidders will be refunded.");
        await loadAuctionInfo();
    } catch (err) { alert("Error: " + err.message); }
});

document.getElementById("finalizeBtn")?.addEventListener("click", async () => {
    try {
        await account.functionCall({
            contractId: CONTRACT_ID, methodName: "finalize",
            args: {}, gas: "100000000000000",
        });
        alert("Auction finalized! Winner's bid sent to auctioneer.");
        await loadAuctionInfo();
    } catch (err) { alert("Error: " + err.message); }
});

document.getElementById("withdrawBtn")?.addEventListener("click", async () => {
    try {
        await account.functionCall({
            contractId: CONTRACT_ID, methodName: "withdraw",
            args: {}, gas: "100000000000000",
        });
        alert("Funds withdrawn successfully!");
    } catch (err) { alert("Error: " + err.message); }
});

// === UTILS ===
function truncateAddress(addr) {
    if (!addr || addr.length <= 20) return addr || "";
    return addr.slice(0, 10) + "..." + addr.slice(-8);
}

function yoctoToNear(yocto) {
    if (!yocto || yocto === "0") return "0";
    const str = yocto.padStart(25, "0");
    const intPart = str.slice(0, str.length - 24) || "0";
    const decPart = str.slice(str.length - 24, str.length - 22);
    return decPart === "00" ? intPart : `${intPart}.${decPart}`;
}

function pad(n) { return n.toString().padStart(2, "0"); }

// === START ===
window.addEventListener("DOMContentLoaded", initNear);
