// App State - LocalStorage Cart
let cart = JSON.parse(localStorage.getItem("cart")) || [];

// DOM Ready Handler
document.addEventListener("DOMContentLoaded", function() {
    initTheme();
    updateCartBadge();
    initCartDrawer();
    
    // Page-specific initializers
    if (document.getElementById("cart-page-items-body")) {
        renderCartPage();
    }
    if (document.getElementById("checkout-items-list")) {
        renderCheckoutSummary();
    }

    // Set click handler on dynamically/statically rendered add-to-cart buttons
    document.body.addEventListener("click", function(e) {
        let btn = e.target.closest(".add-to-cart-btn");
        if (btn) {
            let item = {
                id: btn.getAttribute("data-id"),
                name: btn.getAttribute("data-name"),
                price: parseFloat(btn.getAttribute("data-price")),
                image: btn.getAttribute("data-image"),
                qty: 1
            };
            addToCart(item);
            openCartDrawer();
        }
    });
});

// --- THEME SWITCHER ---
function initTheme() {
    const themeToggle = document.getElementById("theme-toggle");
    const currentTheme = localStorage.getItem("theme") || "light";
    
    document.documentElement.setAttribute("data-theme", currentTheme);
    updateThemeIcon(currentTheme);

    if (themeToggle) {
        themeToggle.addEventListener("click", () => {
            let theme = document.documentElement.getAttribute("data-theme");
            let nextTheme = theme === "dark" ? "light" : "dark";
            
            document.documentElement.setAttribute("data-theme", nextTheme);
            localStorage.setItem("theme", nextTheme);
            updateThemeIcon(nextTheme);
        });
    }
}

function updateThemeIcon(theme) {
    const themeToggle = document.getElementById("theme-toggle");
    if (!themeToggle) return;
    
    const icon = themeToggle.querySelector("i");
    if (theme === "dark") {
        icon.className = "fa-solid fa-sun";
    } else {
        icon.className = "fa-solid fa-moon";
    }
}

// --- CART STATE MANAGER ---
function addToCart(newItem) {
    let existing = cart.find(item => item.id === newItem.id);
    if (existing) {
        existing.qty += newItem.qty;
    } else {
        cart.push(newItem);
    }
    saveCart();
}

function updateCartQty(id, qty) {
    let item = cart.find(i => i.id === id);
    if (item) {
        item.qty = qty;
        if (item.qty <= 0) {
            cart = cart.filter(i => i.id !== id);
        }
    }
    saveCart();
}

function removeFromCart(id) {
    cart = cart.filter(item => item.id !== id);
    saveCart();
}

function saveCart() {
    localStorage.setItem("cart", JSON.stringify(cart));
    updateCartBadge();
    renderCartDrawerItems();
    if (document.getElementById("cart-page-items-body")) {
        renderCartPage();
    }
    if (document.getElementById("checkout-items-list")) {
        renderCheckoutSummary();
    }
}

function getCartTotal() {
    return cart.reduce((total, item) => total + (item.price * item.qty), 0);
}

function getCartCount() {
    return cart.reduce((count, item) => count + item.qty, 0);
}

function updateCartBadge() {
    const badges = [
        document.getElementById("cart-badge-count")
    ];
    let count = getCartCount();
    badges.forEach(b => {
        if (b) {
            b.innerText = count;
            b.style.display = count > 0 ? "block" : "none";
        }
    });
}

// --- CART DRAWER CONTROLS ---
function initCartDrawer() {
    const btn = document.getElementById("cart-button");
    const drawer = document.getElementById("cart-drawer");
    const overlay = document.getElementById("cart-drawer-overlay");
    const closeBtn = document.getElementById("cart-drawer-close");

    if (btn && drawer && overlay && closeBtn) {
        btn.addEventListener("click", function(e) {
            e.preventDefault();
            openCartDrawer();
        });

        closeBtn.addEventListener("click", closeCartDrawer);
        overlay.addEventListener("click", closeCartDrawer);
    }

    renderCartDrawerItems();
}

function openCartDrawer() {
    document.getElementById("cart-drawer").classList.add("active");
    document.getElementById("cart-drawer-overlay").classList.add("active");
    renderCartDrawerItems();
}

function closeCartDrawer() {
    document.getElementById("cart-drawer").classList.remove("active");
    document.getElementById("cart-drawer-overlay").classList.remove("active");
}

function renderCartDrawerItems() {
    const container = document.getElementById("cart-drawer-items");
    const totalLabel = document.getElementById("cart-drawer-total");
    if (!container) return;

    container.innerHTML = "";
    
    if (cart.length === 0) {
        container.innerHTML = `<p class="empty-cart-message">Ваша корзина пуста</p>`;
        if (totalLabel) totalLabel.innerText = "0.00 руб.";
        return;
    }

    cart.forEach(item => {
        let div = document.createElement("div");
        div.className = "drawer-item";
        div.innerHTML = `
            <img src="${item.image || '/static/images/placeholder.jpg'}" alt="${item.name}" class="drawer-item-img">
            <div class="drawer-item-details">
                <div class="drawer-item-title">${item.name}</div>
                <div class="drawer-item-price">${item.qty} x ${item.price} руб.</div>
            </div>
            <button class="drawer-item-remove" onclick="removeFromCart('${item.id}')">
                <i class="fa-solid fa-trash-can"></i>
            </button>
        `;
        container.appendChild(div);
    });

    if (totalLabel) {
        totalLabel.innerText = getCartTotal().toFixed(2) + " руб.";
    }
}

// --- PRODUCT DETAILS PAGE UTILITIES ---
function switchProductImage(src, element) {
    const mainImg = document.getElementById("main-product-img");
    if (mainImg) {
        mainImg.src = src;
    }
    
    // Switch active state border
    const thumbs = document.querySelectorAll(".thumb-item");
    thumbs.forEach(t => t.classList.remove("active"));
    if (element) {
        element.classList.add("active");
    }
}

function adjustProductQty(delta) {
    const input = document.getElementById("product-qty-input");
    if (!input) return;

    let val = parseInt(input.value) + delta;
    let min = parseInt(input.getAttribute("min")) || 1;
    let max = parseInt(input.getAttribute("max")) || 999;

    if (val >= min && val <= max) {
        input.value = val;
    }
}

function addProductWithQty(btn) {
    const qtyInput = document.getElementById("product-qty-input");
    let qty = qtyInput ? parseInt(qtyInput.value) : 1;

    let item = {
        id: btn.getAttribute("data-id"),
        name: btn.getAttribute("data-name"),
        price: parseFloat(btn.getAttribute("data-price")),
        image: btn.getAttribute("data-image"),
        qty: qty
    };
    addToCart(item);
    openCartDrawer();
}

function switchTab(tabId, btn) {
    // Switch Active Button
    const btns = document.querySelectorAll(".tab-header-btn");
    btns.forEach(b => b.classList.remove("active"));
    btn.classList.add("active");

    // Switch Active Pane
    const panes = document.querySelectorAll(".tab-pane");
    panes.forEach(p => p.classList.remove("active"));
    document.getElementById(tabId).classList.add("active");
}

function submitProductReview(e, productID) {
    e.preventDefault();
    
    let author = document.getElementById("review-author").value;
    let rating = parseInt(document.querySelector('input[name="rating"]:checked').value);
    let text = document.getElementById("review-text").value;

    fetch("/api/review", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            product_id: productID,
            author: author,
            rating: rating,
            text: text
        })
    })
    .then(res => res.json())
    .then(data => {
        if (data.status === "success") {
            alert("Спасибо за отзыв! Он опубликован.");
            location.reload();
        } else {
            alert("Не удалось отправить отзыв. Попробуйте еще раз.");
        }
    })
    .catch(err => {
        console.error("Review error:", err);
        alert("Произошла ошибка при отправке.");
    });
}

// --- CART PAGE RENDERER ---
function renderCartPage() {
    const tbody = document.getElementById("cart-page-items-body");
    const table = document.getElementById("cart-page-table");
    const summary = document.getElementById("cart-summary-block");
    const emptyMsg = document.getElementById("cart-empty-message");
    
    if (!tbody) return;
    tbody.innerHTML = "";

    if (cart.length === 0) {
        if (table) table.style.display = "none";
        if (summary) summary.style.display = "none";
        if (emptyMsg) emptyMsg.style.display = "block";
        return;
    }

    if (table) table.style.display = "table";
    if (summary) summary.style.display = "block";
    if (emptyMsg) emptyMsg.style.display = "none";

    cart.forEach(item => {
        let tr = document.createElement("tr");
        tr.innerHTML = `
            <td>
                <div class="cart-item-info">
                    <img src="${item.image}" alt="${item.name}" class="cart-item-image">
                    <span class="cart-item-name">${item.name}</span>
                </div>
            </td>
            <td>${item.price} руб.</td>
            <td>
                <div class="quantity-selector">
                    <button class="qty-btn" onclick="updateCartQty('${item.id}', ${item.qty - 1})">-</button>
                    <input type="text" value="${item.qty}" readonly>
                    <button class="qty-btn" onclick="updateCartQty('${item.id}', ${item.qty + 1})">+</button>
                </div>
            </td>
            <td><strong>${(item.price * item.qty).toFixed(2)} руб.</strong></td>
            <td>
                <button class="drawer-item-remove" onclick="removeFromCart('${item.id}')">
                    <i class="fa-solid fa-trash-can"></i>
                </button>
            </td>
        `;
        tbody.appendChild(tr);
    });

    // Update Summary Box
    document.getElementById("summary-items-count").innerText = getCartCount() + " шт.";
    document.getElementById("summary-total-price").innerText = getCartTotal().toFixed(2) + " руб.";
}

// --- CHECKOUT PAGE LOGIC ---
let activeShippingCost = 0;

function updateShippingCost(cost) {
    activeShippingCost = cost;
    
    // Simulates X-Shipping rule: Courier delivery (pickup excluded) is free for orders over 2000
    let subtotal = getCartTotal();
    const selectedShipping = document.querySelector('input[name="shipping_method"]:checked').value;
    
    if (selectedShipping === "courier" && subtotal >= 2000) {
        activeShippingCost = 0;
        const disp = document.getElementById("courier-price-display");
        if (disp) disp.innerText = "0 руб. (Акция!)";
    } else {
        const disp = document.getElementById("courier-price-display");
        if (disp) disp.innerText = "200 руб.";
    }

    const shipLabel = document.getElementById("checkout-shipping-cost");
    if (shipLabel) shipLabel.innerText = activeShippingCost.toFixed(2) + " руб.";
    
    const totalLabel = document.getElementById("checkout-total-price");
    if (totalLabel) totalLabel.innerText = (subtotal + activeShippingCost).toFixed(2) + " руб.";
}

function renderCheckoutSummary() {
    const list = document.getElementById("checkout-items-list");
    if (!list) return;

    list.innerHTML = "";
    
    if (cart.length === 0) {
        list.innerHTML = `<p>Ваша корзина пуста</p>`;
        return;
    }

    cart.forEach(item => {
        let div = document.createElement("div");
        div.className = "summary-item-inline";
        div.innerHTML = `
            <span>${item.name} x ${item.qty}</span>
            <span>${(item.price * item.qty).toFixed(2)} руб.</span>
        `;
        list.appendChild(div);
    });

    let subtotal = getCartTotal();
    document.getElementById("checkout-subtotal").innerText = subtotal.toFixed(2) + " руб.";
    updateShippingCost(activeShippingCost);
}

function handleCheckoutSubmit(e) {
    e.preventDefault();

    if (cart.length === 0) {
        alert("Ваша корзина пуста!");
        return;
    }

    let name = document.getElementById("checkout-name").value;
    let phone = document.getElementById("checkout-phone").value;
    let email = document.getElementById("checkout-email").value;
    let comment = document.getElementById("checkout-comment").value;
    let shipping = document.querySelector('input[name="shipping_method"]:checked').value;
    let payment = document.querySelector('input[name="payment_method"]:checked').value;

    let payload = {
        name: name,
        phone: phone,
        email: email,
        comment: comment,
        delivery: shipping === "pickup" ? "Самовывоз" : (shipping === "courier" ? "Доставка курьером" : "Доставка почтой"),
        payment: payment === "cod" ? "Наличными при получении" : "LiqPay Онлайн",
        cart: cart.map(item => ({ id: item.id, qty: item.qty, price: item.price }))
    };

    fetch("/api/order", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    })
    .then(res => res.json())
    .then(data => {
        if (data.status === "success") {
            // Hide Form Layout
            document.getElementById("checkout-form-layout").style.display = "none";
            
            // Show Success Screen
            const successScreen = document.getElementById("checkout-success-screen");
            successScreen.style.display = "block";
            
            // Fill order numbers
            document.getElementById("success-order-number").innerText = data.order_number;
            
            if (payment === "liqpay") {
                // Show LiqPay Sandbox Payment Box
                document.getElementById("liqpay-simulation-box").style.display = "block";
                let grandTotal = parseFloat(data.total) + activeShippingCost;
                document.getElementById("liqpay-amount").innerText = grandTotal.toFixed(2) + " руб.";
            }

            // Clear local cart
            cart = [];
            localStorage.removeItem("cart");
            updateCartBadge();
        } else {
            alert("Не удалось оформить заказ. Попробуйте еще раз.");
        }
    })
    .catch(err => {
        console.error("Checkout error:", err);
        alert("Произошла ошибка при оформлении заказа.");
    });
}

function simulateLiqPayPayment() {
    alert("Тестовый платеж LiqPay успешно обработан! Заказ оплачен.");
    document.getElementById("liqpay-simulation-box").style.display = "none";
}
