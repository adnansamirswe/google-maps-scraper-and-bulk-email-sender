<script lang="ts">
	import { onMount } from 'svelte';
	import { apiRequest } from '$lib/api';
	import { alerts } from '$lib/alerts.svelte';

	let smtps = $state<any[]>([]);
	let selectedSmtp = $state<string>('');
	
	let subject = $state('');
	let targets = $state('');
	let isSending = $state(false);
	
	let invalidEntries = $state<string[]>([]);
	let showValidationModal = $state(false);
	
	let isEditing = $state(false);
	let editingId = $state<string | null>(null);
	let showSmtpModal = $state(false);
	let newSmtp = $state({ name: '', host: '', port: 587, username: '', password: '', from_email: '' });
	
	let mirrorDiv: HTMLElement;
	function handleScroll(e: Event) {
		const textarea = e.target as HTMLTextAreaElement;
		if (mirrorDiv) {
			mirrorDiv.scrollTop = textarea.scrollTop;
			mirrorDiv.scrollLeft = textarea.scrollLeft;
		}
	}

	function escapeHtml(text: string) {
		const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' };
		return text.replace(/[&<>"']/g, (m) => map[m as keyof typeof map]);
	}

	let renderHighlightedText = $derived.by(() => {
		if (!targets) return '';
		// Split while keeping delimiters
		const parts = targets.split(/([\n,;]+)/);
		return parts.map(p => {
			if (/[\n,;]+/.test(p)) return escapeHtml(p);
			const trimmed = p.trim();
			if (!trimmed) return escapeHtml(p);
			if (validateEmail(trimmed)) {
				return escapeHtml(p);
			} else {
				return `<mark class="hl-error">${escapeHtml(p)}</mark>`;
			}
		}).join('');
	});
	
	let quill: any;
	let editorContainer: HTMLElement;

	function validateEmail(email: string) {
		const lower = email.toLowerCase();
		// Strict Regex for standard user@domain.tld format
		const re = /^[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/;
		if (!re.test(lower)) return false;

		// Blacklist common media extensions that might masquerade as emails (retina logos, icons)
		const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp', '.bmp', '.tiff'];
		if (imageExtensions.some(ext => lower.endsWith(ext))) return false;

		// Blacklist common retina/logo keywords
		if (lower.includes('@2x') || lower.includes('@3x') || lower.startsWith('logo-') || lower.includes('-logo')) return false;

		return true;
	}

	function checkTargets() {
		const raw = targets.split(/[\n,;]+/).map(e => e.trim()).filter(Boolean);
		if (raw.length === 0) return true;
		
		invalidEntries = raw.filter(e => !validateEmail(e));
		if (invalidEntries.length > 0) {
			showValidationModal = true;
			return false;
		}
		return true;
	}

	function removeInvalid() {
		const raw = targets.split(/[\n,;]+/).map(e => e.trim()).filter(Boolean);
		const valid = raw.filter(e => validateEmail(e));
		targets = valid.join('\n');
		showValidationModal = false;
		invalidEntries = [];
	}
	
	function filterMedia() {
		const raw = targets.split(/[\n,;]+/).map(e => e.trim()).filter(Boolean);
		const filtered = raw.filter(e => {
			const lower = e.toLowerCase();
			const isImage = ['.png', '.jpg', '.jpeg', '.svg', '.webp'].some(ext => lower.endsWith(ext));
			const isLogo = lower.includes('@2x') || lower.includes('@3x') || lower.startsWith('logo-') || lower.includes('-logo');
			return !(isImage || isLogo);
		});
		if (raw.length !== filtered.length) {
			targets = filtered.join('\n');
		} else {
			alert("No image assets detected in the list.");
		}
	}

	let targetsAnalytics = $derived.by(() => {
		const raw = targets.split(/[\n,;]+/).map(e => e.trim()).filter(Boolean);
		if (raw.length === 0) return { total: 0, valid: 0, invalid: 0 };
		
		const valid = raw.filter(e => validateEmail(e)).length;
		return {
			total: raw.length,
			valid,
			invalid: raw.length - valid
		};
	});

	onMount(async () => {
		await loadSMTPs();

		// Dynamically load Quill JS so we don't break Svelte bundlers
		if (!document.getElementById('quill-script')) {
			const link = document.createElement('link');
			link.rel = 'stylesheet';
			link.href = 'https://cdn.quilljs.com/1.3.6/quill.snow.css';
			document.head.appendChild(link);

			const script = document.createElement('script');
			script.id = 'quill-script';
			script.src = 'https://cdn.quilljs.com/1.3.6/quill.min.js';
			script.onload = () => {
				initQuill();
			};
			document.head.appendChild(script);
		} else {
			initQuill();
		}
	});

	function initQuill() {
		if ((window as any).Quill && editorContainer) {
			quill = new (window as any).Quill(editorContainer, {
				theme: 'snow',
				placeholder: 'Compose your elegant HTML email...',
				modules: {
					toolbar: [
						[{ header: [1, 2, false] }],
						['bold', 'italic', 'underline', 'strike', 'blockquote'],
						[{ list: 'ordered' }, { list: 'bullet' }],
						['link', 'image'],
						['clean']
					]
				}
			});
		}
	}

	async function loadSMTPs() {
		try {
			smtps = await apiRequest('/api/smtp');
			if (smtps.length > 0 && !selectedSmtp) {
				selectedSmtp = smtps[0].id;
			}
		} catch (err) {
			console.error("Failed to load SMTP configs", err);
		}
	}

	async function saveSmtp() {
		if (!newSmtp.name || !newSmtp.host || !newSmtp.username || !newSmtp.password) {
			alerts.warning('Please fill out all required SMTP fields.');
			return;
		}
		
		newSmtp.port = Number(newSmtp.port);
		
		try {
			if (isEditing && editingId) {
				await apiRequest(`/api/smtp/${editingId}`, 'PUT', newSmtp);
				alerts.success('SMTP Profile updated successfully!');
			} else {
				await apiRequest('/api/smtp', 'POST', newSmtp);
				alerts.success('SMTP Profile saved successfully!');
			}
			await loadSMTPs();
			cancelSmtpEdit();
		} catch (err: any) {
			alerts.error(`Failed to ${isEditing ? 'update' : 'save'} SMTP: ` + err.message);
		}
	}

	function openAddModal() {
		isEditing = false;
		editingId = null;
		newSmtp = { name: '', host: '', port: 587, username: '', password: '', from_email: '' };
		showSmtpModal = true;
	}

	function openEditModal() {
		const s = smtps.find(x => x.id === selectedSmtp);
		if (!s) return;
		isEditing = true;
		editingId = s.id;
		newSmtp = { ...s };
		showSmtpModal = true;
	}

	function cancelSmtpEdit() {
		showSmtpModal = false;
		isEditing = false;
		editingId = null;
		newSmtp = { name: '', host: '', port: 587, username: '', password: '', from_email: '' };
	}
	
	async function deleteSmtp(id: string) {
		const confirmed = await alerts.confirm('Delete SMTP Profile', 'Are you sure you want to permanently remove this SMTP relay?');
		if (!confirmed) return;
		try {
			await apiRequest(`/api/smtp/${id}`, 'DELETE');
			if (selectedSmtp === id) selectedSmtp = '';
			await loadSMTPs();
			alerts.success('SMTP Profile deleted.');
		} catch(err: any) {
			alerts.error('Failed to delete SMTP: ' + err.message);
		}
	}

	async function sendBlast() {
		if (!selectedSmtp) return alerts.error('Please select an SMTP Configuration.');
		if (!subject) return alerts.error('Please enter an email subject.');
		if (!targets) return alerts.warning('Please paste at least one target email.');
		
		// Run validation check
		if (!checkTargets()) return;

		const bodyHTML = quill ? quill.root.innerHTML : '';
		if (!bodyHTML || bodyHTML === '<p><br></p>') return alerts.warning('Please compose an email body.');

		// Parse emails intelligently from raw pasted data
		// This easily accepts copied data from our history tables (newline/comma separated)
		const emails = targets.split(/[\n,;]+/)
			.map(e => e.trim())
			.filter(e => e.includes('@'));

		if (emails.length === 0) return alerts.error('No valid email addresses found in targets input.');

		const confirmed = await alerts.confirm('Dispatch Bulk Blast', `Are you ready to send ${emails.length} emails to your selected targets?`);
		if (!confirmed) return;

		isSending = true;
		try {
			const res = await apiRequest('/api/emailer/send', 'POST', {
				smtp_id: selectedSmtp,
				subject,
				body_html: bodyHTML,
				emails
			});
			alerts.success(`Bulk Blast Dispatched: ${res.msg}`);
			targets = '';
		} catch (err: any) {
			alerts.error('Failed to initialize bulk blast: ' + err.message);
		} finally {
			isSending = false;
		}
	}
</script>

<div class="page-header">
	<h1>
		<span class="mi mi-lg">mail</span>
		Bulk Email Studio
	</h1>
	<p class="text-secondary">Load your extracted leads and dispatch high-conversion rich text emails directly through your custom SMTP relays.</p>
</div>

<div class="card animate-fade-in-up">
	
	<!-- SMTP Toolbar -->
	<div class="section-header">
		<div class="section-title">
			<span class="mi">router</span> Server Routing
		</div>
		<button class="btn btn-primary btn-sm" onclick={openAddModal}>
			<span class="mi mi-sm">add</span> Add SMTP Server
		</button>
	</div>
	
	<div class="p-6">
		<div class="flex items-center gap-4">
			<select class="input grow" bind:value={selectedSmtp}>
				<option value="">Select a Mail Relay Server...</option>
				{#each smtps as smtp}
					<option value={smtp.id}>{smtp.name} ({smtp.username})</option>
				{/each}
			</select>
			
			{#if selectedSmtp}
				<div class="flex gap-2">
					<button class="btn btn-secondary btn-sm" onclick={openEditModal} title="Edit credentials">
						<span class="mi mi-sm">edit</span> Edit
					</button>
					<button class="btn btn-secondary btn-sm text-red-400" onclick={() => deleteSmtp(selectedSmtp)} title="Remove server">
						<span class="mi mi-sm">delete</span> Remove
					</button>
				</div>
			{/if}
		</div>
	</div>

	<hr style="border-color: var(--border); margin: 24px 0;" />

	<!-- Targets -->
	<div class="mb-6">
		<div class="flex justify-between items-center mb-2">
			<div class="flex items-center gap-3">
				<label class="block text-sm font-semibold m-0">Target Emails List</label>
				{#if targetsAnalytics.total > 0}
					<div class="flex items-center gap-2">
						<span class="chip chip-sm">{targetsAnalytics.total} Total</span>
						{#if targetsAnalytics.invalid > 0}
							<span class="status-badge status-badge-failed px-2 py-0.5 text-xs flex items-center gap-1">
								<span class="mi mi-sm">warning</span>
								{targetsAnalytics.invalid} Invalid
							</span>
						{:else}
							<span class="status-badge status-badge-done px-2 py-0.5 text-xs flex items-center gap-1">
								<span class="mi mi-sm">check_circle</span>
								All Valid
							</span>
						{/if}
					</div>
				{/if}
			</div>
			<div class="flex gap-2">
				{#if targets.trim()}
					<button class="btn btn-secondary btn-sm" onclick={filterMedia} title="Remove logos, pngs, and jpegs">
						<span class="mi mi-sm">image_not_supported</span> Filter Media
					</button>
					<button class="btn btn-secondary btn-sm" onclick={() => checkTargets()}>
						<span class="mi mi-sm">cleaning_services</span> Clean List
					</button>
				{/if}
			</div>
		</div>
		<p class="text-xs text-secondary mb-2">Paste your copied emails here. We automatically separate spaces, commas, and newlines.</p>
		
		<div class="input-highlighter-container">
			<!-- The Backdrop (Highlights) -->
			<div bind:this={mirrorDiv} class="input-highlighter-backdrop">
				{@html renderHighlightedText}
			</div>
			
			<!-- The Actual Input -->
			<textarea 
				class="input w-full font-mono text-sm input-highlighter-textarea"
				class:border-accent={targetsAnalytics.invalid > 0}
				rows="7" 
				placeholder="john@example.com&#10;sarah@company.co.uk"
				bind:value={targets}
				onscroll={handleScroll}></textarea>
		</div>
	</div>

	<!-- Subject -->
	<div class="mb-6">
		<label class="block text-sm font-semibold mb-2">Subject Line</label>
		<input type="text" class="input w-full" placeholder="Introducing our new synergy pipeline..." bind:value={subject} />
	</div>

	<!-- Body -->
	<div class="mb-6">
		<label class="block text-sm font-semibold mb-2">Email Body (Rich Text)</label>
		<div class="quill-wrapper">
			<div bind:this={editorContainer} style="min-height: 250px; background: var(--bg-card); color: var(--text-primary); border: none; border-radius: 0 0 8px 8px;"></div>
		</div>
	</div>

	<div class="flex justify-end mt-4">
		<button class="btn btn-primary" onclick={sendBlast} disabled={isSending || smtps.length === 0}>
			{#if isSending}
				<span class="mi mi-sm animate-spin">sync</span>
			{:else}
				<span class="mi mi-sm">send</span>
			{/if}
			Dispatch Bulk Blast
		</button>
	</div>
</div>

<!-- SMTP Modal -->
{#if showSmtpModal}
<div class="modal-backdrop">
	<div class="modal">
		<div class="modal-header">
			<h3>{isEditing ? 'Edit SMTP Configuration' : 'Add SMTP Configuration'}</h3>
			<button class="btn-icon" onclick={cancelSmtpEdit}>
				<span class="mi">close</span>
			</button>
		</div>
		<div class="modal-body">
			<div class="form-group">
				<label>Profile Name (e.g., Mailgun Main)</label>
				<input type="text" class="input flex-1" bind:value={newSmtp.name} />
			</div>
			<div class="flex gap-4 mb-4">
				<div class="form-group flex-2">
					<label>SMTP Host</label>
					<input type="text" class="input w-full" placeholder="smtp.mailgun.org" bind:value={newSmtp.host} />
				</div>
				<div class="form-group flex-1">
					<label>Port</label>
					<input type="number" class="input w-full" placeholder="587" bind:value={newSmtp.port} />
				</div>
			</div>
			
			<div class="form-group">
				<label>From Email Address</label>
				<input type="email" class="input flex-1" placeholder="sales@yourdomain.com" bind:value={newSmtp.from_email} />
			</div>
			<div class="form-group">
				<label>Username / API Key</label>
				<input type="text" class="input flex-1" bind:value={newSmtp.username} />
			</div>
			<div class="form-group mb-0">
				<label>Password / Secret</label>
				<input type="password" class="input flex-1" bind:value={newSmtp.password} />
			</div>
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary" onclick={cancelSmtpEdit}>Cancel</button>
			<button class="btn btn-primary" onclick={saveSmtp}>
				{isEditing ? 'Update Configuration' : 'Save SMTP Profile'}
			</button>
		</div>
	</div>
</div>
{/if}

<!-- Validation Modal -->
{#if showValidationModal}
<div class="modal-backdrop">
	<div class="modal">
		<div class="modal-header">
			<h3 class="flex items-center gap-2">
				<span class="mi text-accent">warning</span>
				Review Invalid Targets
			</h3>
			<button class="btn-icon" onclick={() => showValidationModal = false}>
				<span class="mi">close</span>
			</button>
		</div>
		<div class="modal-body">
			<p class="text-sm text-secondary mb-4">The following entries do not appear to be valid email addresses. Would you like to remove them or keep the list as is?</p>
			
			<div class="bg-black bg-opacity-20 rounded p-3 font-mono text-xs max-h-48 overflow-y-auto">
				{#each invalidEntries as entry}
					<div class="py-1 border-b border-gray-700 last:border-0">{entry}</div>
				{/each}
			</div>
		</div>
		<div class="modal-footer">
			<button class="btn btn-secondary" onclick={() => showValidationModal = false}>Keep All</button>
			<button class="btn btn-primary" onclick={removeInvalid}>Remove Invalid</button>
		</div>
	</div>
</div>
{/if}

<style>
	.quill-wrapper {
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		overflow: hidden;
	}
	
	/* Override Quill Toolbar for Dark Mode Harmony */
	:global(.ql-toolbar.ql-snow) {
		background: var(--bg-card);
		border-radius: 8px 8px 0 0;
		border: none !important;
		border-bottom: 1px solid var(--border) !important;
	}
	:global(.ql-container.ql-snow) {
		border: none !important;
		font-family: inherit;
		font-size: 1rem;
		background: var(--bg-card);
		color: var(--text-primary);
	}
	:global(.ql-editor.ql-blank::before) {
		color: var(--text-muted);
		font-style: normal;
	}
	:global(.ql-snow .ql-stroke) {
		stroke: var(--text-secondary);
	}
	:global(.ql-snow .ql-fill) {
		fill: var(--text-secondary);
	}
	:global(.ql-snow .ql-picker) {
		color: var(--text-secondary);
	}

	.input-highlighter-container {
		position: relative;
		width: 100%;
		min-height: 150px;
	}

	.input-highlighter-backdrop, .input-highlighter-textarea {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		padding: 12px;
		margin: 0;
		font-family: inherit;
		font-size: 0.825rem;
		line-height: 1.5;
		white-space: pre-wrap;
		word-wrap: break-word;
		border: 1px solid transparent;
		box-sizing: border-box;
	}

	.input-highlighter-backdrop {
		color: transparent;
		z-index: 1;
		pointer-events: none;
		overflow-y: auto;
		background: transparent;
	}

	.input-highlighter-textarea {
		z-index: 2;
		background: transparent !important;
		color: var(--text-primary);
		caret-color: var(--accent-1);
		scrollbar-gutter: stable;
		resize: vertical;
	}

	:global(.hl-error) {
		background: rgba(239, 68, 68, 0.35);
		border-bottom: 2px solid #ef4444;
		border-radius: 2px;
		color: transparent;
	}
</style>
