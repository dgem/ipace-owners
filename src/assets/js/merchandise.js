(function () {
  'use strict';
  var root = document.querySelector('[data-shop]');
  if (!root) return;
  var products = [], basket = [], receipt = '', ready = false, busy = false;
  var status = root.querySelector('[data-shop-status]');
  var form = root.querySelector('[data-shop-form]');
  var quotePanel = root.querySelector('[data-shop-quote-result]');
  var basketPanel = root.querySelector('[data-shop-basket]');
  function esc(value) { var node = document.createElement('span'); node.textContent = String(value == null ? '' : value); return node.innerHTML; }
  function money(pence) { return new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' }).format(pence / 100); }
  function api(path, options) {
    return fetch('/api/shop/' + path, options).then(function (response) {
      return response.json().then(function (data) { if (!response.ok) throw new Error(data.error || 'The shop is temporarily unavailable. Please try again.'); return data; });
    });
  }
  function invalidateQuote() { receipt = ''; quotePanel.hidden = true; quotePanel.innerHTML = ''; }
  function renderBasket() {
    invalidateQuote(); basketPanel.hidden = basket.length === 0;
    root.querySelector('[data-shop-basket-link]').hidden = !basket.length;
    root.querySelector('[data-shop-lines]').innerHTML = basket.map(function (line, i) {
      return '<div class="merch-basket-line"><p><strong>' + esc(line.title) + '</strong><br>' + esc(line.variant) + ' · Quantity: ' + line.quantity + '</p><button type="button" class="btn btn--secondary" data-remove="' + i + '" aria-label="Remove ' + esc(line.title + ' ' + line.variant) + '">Remove</button></div>';
    }).join('');
  }
  function renderProducts() {
    root.querySelector('[data-shop-products]').innerHTML = products.map(function (p, i) {
      var image = p.images.find(function (im) { return im.is_default; }) || p.images[0];
      return '<article class="card merch-product" id="product-' + esc(p.id) + '">' + (image ? '<img src="' + esc(image.src) + '" alt="' + esc(p.title) + '" loading="lazy" width="600" height="600">' : '') + '<div class="merch-product__body stack"><h2>' + esc(p.title) + '</h2><p class="merch-price" data-production-price="' + i + '">From ' + money(p.variants[0].productionPrice) + '</p><label for="shop-variant-' + i + '">Choose option</label><select id="shop-variant-' + i + '" data-variant="' + i + '">' + p.variants.map(function (v) { return '<option value="' + v.id + '">' + esc(v.title) + '</option>'; }).join('') + '</select><details><summary>Product details and sizing</summary><p class="merch-description">' + esc(p.description) + '</p></details><p>Production price; delivered total calculated below.</p><button type="button" class="btn btn--primary" data-add="' + i + '"' + (ready ? '' : ' disabled') + '>Add to basket</button></div></article>';
    }).join('');
  }
  root.addEventListener('click', function (event) {
    var add = event.target.closest('[data-add]'), remove = event.target.closest('[data-remove]');
    if (busy) return;
    if (add) {
      var index = Number(add.dataset.add), p = products[index];
      var id = Number(root.querySelector('[data-variant="' + index + '"]').value);
      var variant = p.variants.find(function (v) { return v.id === id; });
      var count = basket.reduce(function (sum, line) { return sum + line.quantity; }, 0);
      if (count >= 20) { status.textContent = 'Please order no more than 20 items at a time.'; return; }
      var line = basket.find(function (item) { return item.product_id === p.id && item.variant_id === id; });
      if (line) { if (line.quantity >= 10) { status.textContent = 'Maximum 10 of each option per order.'; return; } line.quantity += 1; }
      else { if (basket.length >= 10) { status.textContent = 'Maximum 10 different options per order.'; return; } basket.push({ product_id: p.id, variant_id: id, quantity: 1, title: p.title, variant: variant.title }); }
      renderBasket(); status.textContent = p.title + ' added to your basket. ' + (count + 1) + ' items.';
    }
    if (remove) { basket.splice(Number(remove.dataset.remove), 1); renderBasket(); status.textContent = 'Item removed.'; }
    if (event.target.closest('[data-pay]')) pay();
    if (event.target.closest('[data-order-refresh]')) loadOrder();
  });
  root.addEventListener('change', function (event) {
    if (form.contains(event.target)) invalidateQuote();
    if (event.target.matches('[data-variant]')) {
      var p = products[Number(event.target.dataset.variant)], id = Number(event.target.value);
      var selected = p.variants.find(function (v) { return v.id === id; });
      root.querySelector('[data-production-price="' + event.target.dataset.variant + '"]').textContent = 'From ' + money(selected.productionPrice);
      var image = p.images.find(function (im) { return Array.isArray(im.variant_ids) && im.variant_ids.indexOf(id) !== -1; });
      var node = event.target.closest('article').querySelector('img'); if (image && node) node.src = image.src;
    }
  });
  form.addEventListener('input', invalidateQuote);
  form.addEventListener('submit', function (event) {
    event.preventDefault(); if (busy || !basket.length) return;
    busy = true; invalidateQuote(); root.querySelector('[data-shop-feedback]').textContent = ''; status.textContent = 'Checking availability and delivery…';
    var data = new FormData(form), address = {};
    ['first_name', 'last_name', 'email', 'phone', 'country', 'region', 'address1', 'address2', 'city', 'zip'].forEach(function (key) { address[key] = String(data.get(key) || '').trim(); });
    var button = root.querySelector('[data-shop-quote]'); button.disabled = true;
    // Freeze delivery fields while the quote is requested so the quote cannot
    // silently refer to an address the shopper has already changed.
    var fields = Array.from(form.elements); fields.forEach(function (field) { field.disabled = true; });
    api('quote', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ line_items: basket.map(function (line) { return { product_id: line.product_id, variant_id: line.variant_id, quantity: line.quantity }; }), address_to: address, terms: data.get('terms') === 'on', website: data.get('website') || '' }) }).then(function (quote) {
      receipt = quote.receipt;
      try { sessionStorage.setItem('ipace-shop-receipt-' + quote.orderId, receipt); } catch { /* Checkout still works; email is the fallback receipt. */ }
      quotePanel.innerHTML = '<div class="callout__content"><h3>Your delivered price: ' + money(quote.totals.total) + '</h3><p>' + esc(quote.summary) + '</p><p>Deliver to ' + esc(address.first_name + ' ' + address.last_name + ', ' + address.address1 + ', ' + address.city + ', ' + address.zip + ', ' + address.country) + '.</p><p>Includes production, delivery, supplier-tax allowance and payment costs. Any surplus helps cover the site’s running costs. Import charges collected by your carrier are not included.</p><p>Quote valid for 30 minutes. Pay securely with Stripe; card details are never handled by this website.</p><button type="button" class="btn btn--primary" data-pay>Pay ' + money(quote.totals.total) + ' securely</button></div>';
      quotePanel.hidden = false; quotePanel.focus(); status.textContent = 'Your delivered price is ready. Review the address and total before paying.';
    }).catch(function (error) { status.textContent = error.message; root.querySelector('[data-shop-feedback]').textContent = error.message; }).finally(function () { busy = false; fields.forEach(function (field) { field.disabled = false; }); });
  });
  function pay() {
    if (!receipt || busy) return;
    busy = true; var button = root.querySelector('[data-pay]'); button.disabled = true;
    var previousError = quotePanel.querySelector('[role="alert"]'); if (previousError) previousError.remove();
    api('checkout', { method: 'POST', headers: { Authorization: 'Bearer ' + receipt } }).then(function (data) {
      var target = new URL(data.url); if (target.protocol !== 'https:' || target.hostname !== 'checkout.stripe.com') throw new Error('Invalid checkout link. Please contact us.');
      window.location.assign(target.href);
    }).catch(function (error) { status.textContent = error.message; quotePanel.insertAdjacentHTML('beforeend', '<p role="alert">' + esc(error.message) + '</p>'); busy = false; button.disabled = false; });
  }
  function loadOrder() {
    var saved = ''; try { saved = sessionStorage.getItem('ipace-shop-receipt-' + new URLSearchParams(window.location.search).get('order')) || ''; } catch { /* No saved browser receipt. */ }
    var panel = root.querySelector('[data-shop-order]'); panel.hidden = false;
    if (!saved) { panel.textContent = 'Your payment receipt will arrive by email. Contact us if you need help with your order.'; return; }
    api('order', { headers: { Authorization: 'Bearer ' + saved } }).then(function (order) {
      var labels = { 'test-paid': 'Test payment recorded — no merchandise will be made', quoted: 'Waiting for payment confirmation', submitting: 'Payment received — preparing your order', 'needs-review': 'Payment received — we are checking fulfilment', submitted: 'Order sent to Printify', 'on-hold': 'Order awaiting production', 'sending-to-production': 'Preparing for production', 'in-production': 'Your merchandise is being made', fulfilled: 'Your order has shipped', canceled: 'Fulfilment cancelled — contact us about your payment', 'has-issues': 'Printify has reported an issue — contact us for help' };
      panel.innerHTML = '<h2>' + esc(labels[order.status] || 'Order update pending') + '</h2><p>Reference: ' + esc(order.reference) + '</p><p>' + esc(order.summary) + '</p><p>Total: ' + money(order.totals.total) + '</p><p>Printify will email shipping updates. For help, <a href="/contact/">contact the group</a> with your reference.</p><button class="btn btn--secondary" type="button" data-order-refresh>Refresh order status</button>';
      panel.focus();
    }).catch(function (error) { panel.textContent = error.message; });
  }
  if (new URLSearchParams(window.location.search).has('order')) loadOrder();
  api('catalogue').then(function (data) {
    products = data.products; ready = data.checkoutEnabled;
    root.querySelector('[data-shop-seller]').textContent = data.sellerName && data.sellerAddress ? 'Seller: ' + data.sellerName + ' — ' + data.sellerAddress : '';
    var countries = form.querySelector('[name="country"]');
    data.countries.forEach(function (country) { var option = document.createElement('option'); option.value = country.code; option.textContent = country.name; countries.appendChild(option); });
    root.querySelector('[data-shop-test]').hidden = !data.testMode;
    renderProducts(); status.textContent = ready ? (products.length ? 'Choose your merchandise below. No sign-in needed.' : 'New merchandise is on its way. Please check back soon.') : 'You can browse merchandise now. Checkout is being prepared.';
    if (new URLSearchParams(window.location.search).has('cancelled')) status.textContent = 'Checkout was cancelled. You can choose your merchandise again when you are ready.';
  }).catch(function (error) { status.textContent = error.message; });
}());
