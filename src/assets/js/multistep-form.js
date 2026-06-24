/**
 * multistep-form.js — Generic multi-step form controller.
 *
 * Works entirely from data attributes — not hardcoded to any single form.
 * Attaches to any element with [data-multistep-form].
 *
 * HTML structure expected:
 *   <form data-multistep-form>
 *     <div class="progress-indicator" data-progress>…</div>
 *     <div data-step="1">…step content…</div>
 *     <div data-step="2">…</div>
 *     …
 *     <div class="step-nav">
 *       <button type="button" data-prev>Back</button>
 *       <button type="button" data-next>Next</button>
 *       <button type="submit" data-submit>Submit</button>
 *     </div>
 *   </form>
 *
 * Progressive enhancement: without JS all steps remain visible in a
 * single-page layout. With JS, steps are revealed one at a time.
 */

(function () {
  'use strict';

  var REDUCED_MOTION = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  function initMultiStepForm(form) {
    var steps = Array.from(form.querySelectorAll('[data-step]'));
    if (!steps.length) return;

    var total       = steps.length;
    var currentStep = 0; // 0-indexed

    // Progress bar elements
    var progressFill = form.querySelector('[data-progress-fill]');
    var progressText = form.querySelector('[data-progress-text]');
    var progressStep = form.querySelector('[data-progress-step]');

    // Nav buttons
    var prevBtns  = Array.from(form.querySelectorAll('[data-prev]'));
    var nextBtns  = Array.from(form.querySelectorAll('[data-next]'));
    var submitBtn = form.querySelector('[data-submit]');
    var conditionalSubmitBtns = Array.from(form.querySelectorAll('[data-enable-when-checked]'));

    // Result area
    var resultEl = form.querySelector('[data-submit-result]');

    // ── Initial state (JS enabled) ──────────────────────────────────────────
    setDateMaximums();
    steps.forEach(function (step, i) {
      if (i !== 0) {
        step.classList.add('step--hidden');
        // Disable focusable elements in hidden steps
        setInertState(step, true);
      }
    });

    updateConditionalSubmitControls();
    updateSoftWarnings();
    updateUI();

    // ── Navigation handlers ─────────────────────────────────────────────────
    prevBtns.forEach(function (prevBtn) {
      prevBtn.addEventListener('click', function () {
        if (currentStep > 0) {
          goToStep(currentStep - 1);
        }
      });
    });

    nextBtns.forEach(function (nextBtn) {
      nextBtn.addEventListener('click', function () {
        if (validateCurrentStep() && currentStep < total - 1) {
          goToStep(currentStep + 1);
        }
      });
    });

    // Prevent real submission; show placeholder message instead
    form.addEventListener('submit', function (e) {
      e.preventDefault();

      updateConditionalSubmitControls();
      if (!conditionalSubmitControlsMet()) {
        focusFirstMissingCheckedRequirement();
        return;
      }
      if (!validateCurrentStep()) return;

      if (resultEl) {
        resultEl.classList.add('is-visible');
        resultEl.focus();
        // Hide step nav
        form.querySelectorAll('.step-nav').forEach(function (nav) {
          nav.style.display = 'none';
        });
        // Hide all steps
        steps.forEach(function (s) {
          s.classList.add('step--hidden');
          setInertState(s, true);
        });
        form.dispatchEvent(new CustomEvent('multistep:submitted', {
          bubbles: true,
          detail: {
            form: form,
            result: resultEl
          }
        }));
      } else {
        alert('Submission storage is not enabled yet. Your data has not been sent.');
      }
    });

    form.addEventListener('change', function () {
      updateConditionalSubmitControls();
      updateSoftWarnings();
    });
    form.addEventListener('input', function () {
      updateConditionalSubmitControls();
      updateSoftWarnings();
    });

    // ── Core: go to step ────────────────────────────────────────────────────
    function goToStep(index) {
      if (index < 0 || index >= total) return;

      // Hide current
      steps[currentStep].classList.add('step--hidden');
      setInertState(steps[currentStep], true);

      currentStep = index;

      // Show next
      steps[currentStep].classList.remove('step--hidden');
      setInertState(steps[currentStep], false);

      updateUI();
      moveFocusToStepHeading(steps[currentStep]);

      // Most forms now fit comfortably on desktop. Keep scroll movement opt-in
      // for unusually long flows where step changes would otherwise be confusing.
      if (form.hasAttribute('data-scroll-on-step-change')) {
        if (!REDUCED_MOTION) {
          form.scrollIntoView({ behavior: 'smooth', block: 'start' });
        } else {
          form.scrollIntoView({ block: 'start' });
        }
      }
    }

    // ── Update progress & buttons ────────────────────────────────────────────
    function updateUI() {
      var humanStep  = currentStep + 1;
      var pct        = Math.round((humanStep / total) * 100);

      if (progressFill) {
        progressFill.style.width = pct + '%';
        progressFill.setAttribute('aria-valuenow', String(pct));
      }
      if (progressText) {
        progressText.textContent = 'Step ' + humanStep + ' of ' + total;
      }
      if (progressStep) {
        progressStep.textContent = humanStep;
      }

      // Prev button
      prevBtns.forEach(function (prevBtn) {
        prevBtn.disabled = currentStep === 0;
        prevBtn.style.visibility = currentStep === 0 ? 'hidden' : '';
      });

      // Next vs Submit
      var isLast = currentStep === total - 1;
      nextBtns.forEach(function (nextBtn) {
        nextBtn.style.display = isLast ? 'none' : '';
      });
      if (submitBtn) submitBtn.style.display = isLast ? '' : 'none';
      updateConditionalSubmitControls();
    }

    function updateConditionalSubmitControls() {
      conditionalSubmitBtns.forEach(function (button) {
        button.disabled = !checkedRequirementsMet(button);
      });
    }

    function conditionalSubmitControlsMet() {
      return conditionalSubmitBtns.every(checkedRequirementsMet);
    }

    function focusFirstMissingCheckedRequirement() {
      for (var i = 0; i < conditionalSubmitBtns.length; i++) {
        var raw = conditionalSubmitBtns[i].getAttribute('data-enable-when-checked') || '';
        var names = raw.split(/[\s,]+/).filter(Boolean);
        for (var j = 0; j < names.length; j++) {
          var missing = Array.from(form.elements).find(function (control) {
            return control.name === names[j] && control.type === 'checkbox' && !control.checked;
          });
          if (missing) {
            missing.focus();
            return;
          }
        }
      }
    }

    function checkedRequirementsMet(button) {
      var raw = button.getAttribute('data-enable-when-checked') || '';
      var names = raw.split(/[\s,]+/).filter(Boolean);
      if (!names.length) return true;

      return names.every(function (name) {
        return Array.from(form.elements).some(function (control) {
          return control.name === name && control.type === 'checkbox' && control.checked;
        });
      });
    }

    // ── Validation (light, accessible) ─────────────────────────────────────
    function validateCurrentStep() {
      var step = steps[currentStep];
      var inputs = Array.from(step.querySelectorAll('input, select, textarea'));
      var valid = true;

      inputs.forEach(function (input) {
        // Clear previous error
        input.removeAttribute('aria-invalid');
        var errorEl = input.parentNode.querySelector('[role="alert"]');

        if (isRequiredMissing(input)) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) {
            errorEl.hidden = false;
            errorEl.textContent = 'This field is required.';
          }
          if (valid) {
            // Focus the first invalid field
            input.focus();
          }
          valid = false;
        } else if (input.type === 'email' && input.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.value)) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) {
            errorEl.hidden = false;
            errorEl.textContent = 'Please enter a valid email address.';
          }
          if (valid) input.focus();
          valid = false;
        } else {
          if (errorEl && !input.hasAttribute('data-not-future') && !input.hasAttribute('data-vin-identifier')) {
            errorEl.textContent = '';
          }
        }
      });

      if (!validateRequiredGroups(step)) {
        valid = false;
      }
      if (!validateVINIdentifiers(step)) {
        valid = false;
      }
      if (!validateNotFutureDates(step)) {
        valid = false;
      }

      updateSoftWarnings();

      return valid;
    }

    function setDateMaximums() {
      var today = localDateString(new Date());
      form.querySelectorAll('input[type="date"][data-not-future]').forEach(function (input) {
        input.max = today;
      });
    }

    function localDateString(date) {
      var year = date.getFullYear();
      var month = String(date.getMonth() + 1).padStart(2, '0');
      var day = String(date.getDate()).padStart(2, '0');
      return year + '-' + month + '-' + day;
    }

    function isFutureDate(value) {
      if (!value) return false;
      return value > localDateString(new Date());
    }

    function validateNotFutureDates(step) {
      var valid = true;
      step.querySelectorAll('input[type="date"][data-not-future]').forEach(function (input) {
        var errorEl = input.parentNode.querySelector('[role="alert"]');
        var future = isFutureDate(input.value);
        if (future) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) errorEl.hidden = false;
          if (valid) input.focus();
          valid = false;
        } else if (errorEl) {
          errorEl.hidden = true;
        }
      });
      return valid;
    }

    function normalizeVIN(value) {
      return String(value || '').toUpperCase().replace(/[\s-]/g, '');
    }

    function vinLooksValid(value) {
      return /^[A-HJ-NPR-Z0-9]{17}$/.test(normalizeVIN(value));
    }

    function validateVINIdentifiers(step) {
      var valid = true;
      step.querySelectorAll('[data-vin-identifier]').forEach(function (input) {
        var vin = normalizeVIN(input.value);
        var optionalWith = input.getAttribute('data-vin-optional-with') || '';
        var other = optionalWith && form.elements[optionalWith] ? String(form.elements[optionalWith].value || '').trim() : '';
        var errorEl = input.parentNode.querySelector('[role="alert"]');
        var invalidBlockingVIN = vin && !vinLooksValid(vin) && !other;

        if (invalidBlockingVIN) {
          input.setAttribute('aria-invalid', 'true');
          if (errorEl) errorEl.hidden = false;
          if (valid) input.focus();
          valid = false;
        } else {
          input.removeAttribute('aria-invalid');
          if (errorEl) errorEl.hidden = true;
        }
      });
      return valid;
    }

    function updateSoftWarnings() {
      updateVINWarnings();
      updateUKRegistrationWarnings();
    }

    function updateVINWarnings() {
      form.querySelectorAll('[data-vin-identifier]').forEach(function (input) {
        var warningEl = input.parentNode.querySelector('[data-vin-warning]');
        if (!warningEl) return;
        var vin = normalizeVIN(input.value);
        warningEl.hidden = !(vinLooksValid(vin) && !/^SAD/.test(vin));
      });
    }

    function normalizeRegistration(value) {
      return String(value || '').toUpperCase().replace(/\s+/g, '');
    }

    function ukRegistrationLooksPlausible(value) {
      var reg = normalizeRegistration(value);
      if (!reg) return true;
      return /^[A-Z]{2}[0-9]{2}[A-Z]{3}$/.test(reg) ||
        /^[A-Z][0-9]{1,3}[A-Z]{3}$/.test(reg) ||
        /^[A-Z]{3}[0-9]{1,3}[A-Z]$/.test(reg) ||
        /^[A-Z]{1,3}[0-9]{1,4}$/.test(reg) ||
        /^[0-9]{1,4}[A-Z]{1,3}$/.test(reg) ||
        /^[A-Z]{1,3}[0-9]{1,4}[A-Z]{0,1}$/.test(reg);
    }

    function updateUKRegistrationWarnings() {
      form.querySelectorAll('[data-uk-registration]').forEach(function (input) {
        var warningEl = input.parentNode.querySelector('[data-uk-registration-warning]');
        if (!warningEl) return;
        var countryName = input.getAttribute('data-country-field') || '';
        var country = countryName && form.elements[countryName] ? form.elements[countryName].value : '';
        var shouldCheck = country === 'GB';
        warningEl.hidden = !(shouldCheck && input.value.trim() && !ukRegistrationLooksPlausible(input.value));
      });
    }

    function validateRequiredGroups(step) {
      var groups = Array.from(step.querySelectorAll('[data-require-one]'));
      var valid = true;

      groups.forEach(function (group) {
        var names = (group.getAttribute('data-require-one') || '').split(/[\s,]+/).filter(Boolean);
        var errorEl = group.querySelector('[data-require-one-error]');
        var controls = names.map(function (name) {
          return form.elements[name];
        }).filter(Boolean);
        var hasValue = controls.some(function (control) {
          if (typeof RadioNodeList !== 'undefined' && control instanceof RadioNodeList) {
            return Array.from(control).some(function (item) { return !!String(item.value || '').trim(); });
          }
          if (control.type === 'checkbox' || control.type === 'radio') {
            return control.checked;
          }
          return !!String(control.value || '').trim();
        });

        controls.forEach(function (control) {
          if (typeof RadioNodeList !== 'undefined' && control instanceof RadioNodeList) {
            Array.from(control).forEach(function (item) {
              item.setAttribute('aria-invalid', hasValue ? 'false' : 'true');
            });
          } else if (hasValue) {
            control.removeAttribute('aria-invalid');
          } else {
            control.setAttribute('aria-invalid', 'true');
          }
        });

        if (errorEl) errorEl.hidden = hasValue;
        if (!hasValue) {
          valid = false;
          var first = typeof RadioNodeList !== 'undefined' && controls[0] instanceof RadioNodeList ? controls[0][0] : controls[0];
          if (first && document.activeElement !== first) first.focus();
        }
      });

      return valid;
    }

    function isRequiredMissing(input) {
      if (!input.required) return false;

      if (input.type === 'checkbox') {
        return !input.checked;
      }

      if (input.type === 'radio') {
        return !form.querySelector('input[type="radio"][name="' + input.name + '"]:checked');
      }

      return !input.value.trim();
    }

    // ── Accessibility helpers ────────────────────────────────────────────────
    function moveFocusToStepHeading(stepEl) {
      var heading = stepEl.querySelector('[data-step-heading], h2, h3');
      if (heading) {
        heading.setAttribute('tabindex', '-1');
        heading.focus({ preventScroll: true });
      }
    }

    // Disable / enable focusable elements in a hidden step to prevent
    // keyboard users from tabbing into off-screen content.
    function setInertState(stepEl, inert) {
      var focusable = stepEl.querySelectorAll(
        'input, select, textarea, button, a[href]'
      );
      focusable.forEach(function (el) {
        if (inert) {
          el.setAttribute('tabindex', '-1');
        } else {
          el.removeAttribute('tabindex');
        }
      });
    }
  }

  // ── Bootstrap all forms on the page ────────────────────────────────────────
  document.querySelectorAll('[data-multistep-form]').forEach(initMultiStepForm);

})();
