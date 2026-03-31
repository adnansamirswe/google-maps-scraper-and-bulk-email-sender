<script lang="ts">
	import { onDestroy } from 'svelte';
	import { createScrapeJob, streamJob, getJob, getPlaces } from '$lib/api';
	import { exportToCSV } from '$lib/utils';
	import { alerts } from '$lib/alerts.svelte';

	// Field toggles
	let fields = $state({
		phone: true,
		email: true,
		website: true,
		review_count: true,
		review_rating: true,
		address: true,
		category: true,
		open_hours: false,
		price_range: false,
		status: false,
		coordinates: false,
		description: false,
		reviews_per_rating: true,
	});

	// Form state
	let name = $state('');
	let keywords = $state('');
	let language = $state('en');
	let maxDepth = $state(10);
	let zoom = $state(15);
	let geo = $state('');
	let proxies = $state('');
	let webhookUrl = $state('');
	let concurrency = $state(1);
	let maxResults = $state<number | ''>('');
	let showAdvanced = $state(false);
	let loading = $state(false);
	let error = $state('');

	// Real-time state
	let activeJob = $state<any>(null);
	let liveResults = $state<any[]>([]);
	let streamStatus = $state('');
	let cleanupStream: (() => void) | null = null;
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	// Field definitions for UI
	const fieldDefs = [
		{ key: 'phone', icon: 'phone', label: 'Phone Number' },
		{ key: 'email', icon: 'email', label: 'Email Address' },
		{ key: 'website', icon: 'language', label: 'Website' },
		{ key: 'review_count', icon: 'rate_review', label: 'Review Count' },
		{ key: 'review_rating', icon: 'star', label: 'Average Rating' },
		{ key: 'address', icon: 'location_on', label: 'Address' },
		{ key: 'category', icon: 'category', label: 'Category' },
		{ key: 'open_hours', icon: 'schedule', label: 'Open Hours' },
		{ key: 'price_range', icon: 'attach_money', label: 'Price Range' },
		{ key: 'status', icon: 'info', label: 'Business Status' },
		{ key: 'coordinates', icon: 'my_location', label: 'GPS Coordinates' },
		{ key: 'description', icon: 'description', label: 'Description' },
		{ key: 'reviews_per_rating', icon: 'bar_chart', label: 'Rating Breakdown' },
	] as const;

	// Presets
	function applyPreset(preset: string) {
		for (const key of Object.keys(fields)) {
			(fields as any)[key] = false;
		}

		if (preset === 'contact') {
			fields.phone = true;
			fields.email = true;
			fields.website = true;
		} else if (preset === 'reviews') {
			fields.review_count = true;
			fields.review_rating = true;
			fields.category = true;
		} else if (preset === 'full') {
			for (const key of Object.keys(fields)) {
				(fields as any)[key] = true;
			}
		} else if (preset === 'leads') {
			fields.phone = true;
			fields.email = true;
			fields.website = true;
			fields.address = true;
			fields.category = true;
			fields.review_rating = true;
		}
	}

	function toggleField(key: string) {
		(fields as any)[key] = !(fields as any)[key];
	}

	// Poll job status as fallback in case SSE misses the "done" event
	function startPolling(jobId: string) {
		pollInterval = setInterval(async () => {
			try {
				const job = await getJob(jobId);
				if (job.status === 'done' || job.status === 'failed') {
					stopPolling();
					loading = false;

					if (job.status === 'done') {
						streamStatus = `✅ Done! ${job.places_found} places found`;
						// Fetch all places from API as safety net
						const places = await getPlaces(jobId);
						if (places.length > liveResults.length) {
							liveResults = places;
						}
					} else {
						streamStatus = `❌ Failed: ${job.error_msg || 'Unknown error'}`;
					}
				} else {
					// Update place count from polling
					streamStatus = `Found ${job.places_found} places...`;
				}
			} catch {
				// Ignore polling errors
			}
		}, 3000);
	}

	function stopPolling() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		const keywordList = keywords.split('\n').map(k => k.trim()).filter(Boolean);
		if (keywordList.length === 0) {
			alerts.warning('Please enter at least one keyword');
			return;
		}

		loading = true;
		liveResults = [];
		streamStatus = 'Starting scraper engine...';

		try {
			const proxyList = proxies.split('\n').map(p => p.trim()).filter(Boolean);

			const job = await createScrapeJob({
				name: name || keywordList[0],
				keywords: keywordList,
				language,
				max_depth: maxDepth,
				zoom,
				geo: geo || undefined,
				proxies: proxyList.length > 0 ? proxyList : undefined,
				extract_email: fields.email,
				webhook_url: webhookUrl || undefined,
				concurrency: concurrency || 1,
				max_results: typeof maxResults === 'number' ? maxResults : 0,
				fields,
			});

			activeJob = job;

			// Start polling as a fallback
			startPolling(job.id);

			// Subscribe to real-time SSE updates
			cleanupStream = streamJob(job.id, (event: any) => {
				if (event.type === 'place') {
					liveResults = [...liveResults, event.place];
					streamStatus = `Found ${event.total} places...`;
				} else if (event.type === 'progress') {
					streamStatus = event.msg;
				} else if (event.type === 'done') {
					streamStatus = `✅ Done! ${event.total} places found`;
					loading = false;
					stopPolling();
				} else if (event.type === 'error') {
					streamStatus = `❌ Error: ${event.msg}`;
					loading = false;
					stopPolling();
				}
			});
		} catch (err: any) {
			alerts.error(err.message);
			loading = false;
		}
	}

	function resetForm() {
		if (cleanupStream) cleanupStream();
		stopPolling();
		activeJob = null;
		liveResults = [];
		streamStatus = '';
		loading = false;
	}

	let copyStatus = $state<Record<string, boolean>>({});

	async function copyColumn(id: string, places: any[], column: string) {
		const data = places.map(p => p[column]).filter(Boolean).join('\n');
		if (!data) {
			alerts.warning(`No valid ${column}s available to copy!`);
			return;
		}
		try {
			if (navigator.clipboard && navigator.clipboard.writeText) {
				await navigator.clipboard.writeText(data);
			} else {
				const textArea = document.createElement("textarea");
				textArea.value = data;
				document.body.appendChild(textArea);
				textArea.select();
				document.execCommand("copy");
				textArea.remove();
			}
			copyStatus[id] = true;
			alerts.success(`Copied ${column}s to clipboard!`);
			setTimeout(() => { copyStatus[id] = false; }, 2000);
		} catch (err) {
			console.error('Failed to copy', err);
			alerts.error('Failed to copy to clipboard.');
		}
	}

	onDestroy(() => {
		if (cleanupStream) cleanupStream();
		stopPolling();
	});

	// Computed
	let selectedFieldCount = $derived(Object.values(fields).filter(Boolean).length);
</script>

<div class="page-header">
	<h1>
		<span class="mi mi-lg">search</span>
		New Scrape
	</h1>
	<p class="text-secondary">Configure what data you want to extract from Google Maps</p>
</div>


{#if activeJob}
	<!-- Live Results View -->
	<div class="live-results animate-fade-in-up">
		<div class="live-header">
			<div class="flex items-center gap-3">
				<h2>
					<span class="mi">stream</span>
					Live Results
				</h2>
				<span class="badge" class:badge-running={loading} class:badge-done={!loading}>
					{loading ? 'Scraping' : 'Complete'}
				</span>
			</div>
			<div class="flex items-center flex-wrap gap-2">
				<span class="text-secondary text-sm mr-2">{streamStatus}</span>
				{#if liveResults.length > 0}
					<button class="btn btn-secondary btn-sm" onclick={(e) => copyColumn('live_email', liveResults, 'email')}>
						<span class="mi mi-sm">{copyStatus['live_email'] ? 'check' : 'content_copy'}</span>
						{copyStatus['live_email'] ? 'Copied' : 'Emails'}
					</button>
					<button class="btn btn-secondary btn-sm" onclick={(e) => copyColumn('live_phone', liveResults, 'phone')}>
						<span class="mi mi-sm">{copyStatus['live_phone'] ? 'check' : 'content_copy'}</span>
						{copyStatus['live_phone'] ? 'Copied' : 'Phones'}
					</button>
					<button class="btn btn-secondary btn-sm" onclick={() => exportToCSV(`scrape_${activeJob.id}.csv`, liveResults)}>
						<span class="mi mi-sm">download</span>
						Export CSV
					</button>
				{/if}
				<button class="btn btn-secondary btn-sm" onclick={resetForm}>
					<span class="mi mi-sm">refresh</span>
					New Scrape
				</button>
			</div>
		</div>

		{#if loading}
			<div class="progress-bar progress-bar-indeterminate">
				<div class="progress-bar-fill"></div>
			</div>
		{/if}

		<div class="results-table-wrapper">
			<table class="data-table">
				<thead>
					<tr>
						<th>#</th>
						<th>Business Name</th>
						{#if fields.category}<th>Category</th>{/if}
						{#if fields.phone}<th>Phone</th>{/if}
						{#if fields.email}<th>Email</th>{/if}
						{#if fields.website}<th>Website</th>{/if}
						{#if fields.review_rating}<th>Rating</th>{/if}
						{#if fields.reviews_per_rating}<th>Breakdown</th>{/if}
						{#if fields.review_count}<th>Reviews</th>{/if}
						{#if fields.address}<th>Address</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each liveResults as place, i}
						<tr class="new-row">
							<td class="text-muted">{i + 1}</td>
							<td class="title-cell">
								<strong>{place.title}</strong>
							</td>
							{#if fields.category}
								<td><span class="chip">{place.category || '—'}</span></td>
							{/if}
							{#if fields.phone}
								<td class="font-mono text-sm">{place.phone || '—'}</td>
							{/if}
							{#if fields.email}
								<td class="text-accent text-sm">
									{#if place.email}
										<div class="truncate-clickable" onclick={(e) => e.currentTarget.classList.toggle('expanded')} title={place.email}>{place.email}</div>
									{:else}
										—
									{/if}
								</td>
							{/if}
							{#if fields.website}
								<td>
									{#if place.website}
										<div class="truncate-clickable" onclick={(e) => e.currentTarget.classList.toggle('expanded')} title={place.website}>
											<a href={place.website} target="_blank" class="link">{place.website.replace('https://', '')}</a>
										</div>
									{:else}
										—
									{/if}
								</td>
							{/if}
							{#if fields.review_rating}
								<td>
									{#if place.review_rating}
										<span class="rating">
											<span class="mi mi-sm" style="color: #f39c12;">star</span>
											{place.review_rating.toFixed(1)}
										</span>
									{:else}
										—
									{/if}
								</td>
							{/if}
							{#if fields.reviews_per_rating}
								<td>
									{#if place.reviews_per_rating && Object.keys(place.reviews_per_rating).length > 0}
										<div class="flex items-center gap-2">
											{#each [5,4,3,2,1] as stars}
												{#if place.reviews_per_rating[stars]}
													<span class="text-xs text-secondary bg-gray-100 px-1 rounded dark:bg-gray-800" title="{stars} Stars">{stars}★:{place.reviews_per_rating[stars]}</span>
												{/if}
											{/each}
										</div>
									{:else}
										—
									{/if}
								</td>
							{/if}
							{#if fields.review_count}
								<td class="text-secondary">{place.review_count || 0}</td>
							{/if}
							{#if fields.address}
								<td>
									{#if place.address}
										<div class="truncate-clickable text-sm text-secondary" style="max-width: 180px;" onclick={(e) => e.currentTarget.classList.toggle('expanded')} title={place.address}>{place.address}</div>
									{:else}
										—
									{/if}
								</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{:else}
	<!-- Scrape Configuration Form -->
	<form onsubmit={handleSubmit} class="scrape-form animate-fade-in-up">
		<!-- Keywords -->
		<div class="card">
			<h2 class="section-title">
				<span class="mi">edit_note</span>
				Search Keywords
			</h2>
			<p class="text-secondary text-sm mb-4">Enter one keyword per line (e.g., "restaurants in New York")</p>

			<div class="input-group">
				<label for="name-input">Job Name (optional)</label>
				<input id="name-input" type="text" class="input" placeholder="My scrape job" bind:value={name} />
			</div>

			<div class="input-group mt-4">
				<label for="keywords-input">Keywords</label>
				<textarea
					id="keywords-input"
					class="input"
					placeholder="restaurants in Manhattan&#10;coffee shops in Brooklyn&#10;gyms in Queens"
					bind:value={keywords}
					rows="5"
				></textarea>
			</div>
		</div>

		<!-- Field Selection -->
		<div class="card">
			<div class="section-header">
				<div>
					<h2 class="section-title">
						<span class="mi">tune</span>
						Data Fields
					</h2>
					<p class="text-secondary text-sm">Select which data to extract — {selectedFieldCount} fields selected</p>
				</div>
			</div>

			<!-- Presets -->
			<div class="presets">
				<button type="button" class="preset-chip" onclick={() => applyPreset('contact')}>
					<span class="mi mi-sm">contacts</span> Contact Info
				</button>
				<button type="button" class="preset-chip" onclick={() => applyPreset('reviews')}>
					<span class="mi mi-sm">star</span> Reviews Only
				</button>
				<button type="button" class="preset-chip" onclick={() => applyPreset('leads')}>
					<span class="mi mi-sm">leaderboard</span> Lead Gen
				</button>
				<button type="button" class="preset-chip" onclick={() => applyPreset('full')}>
					<span class="mi mi-sm">select_all</span> All Fields
				</button>
			</div>

			<!-- Field toggles grid -->
			<div class="fields-grid">
				{#each fieldDefs as field}
					<button
						type="button"
						class="field-toggle"
						class:active={(fields as any)[field.key]}
						onclick={() => toggleField(field.key)}
					>
						<span class="mi">{field.icon}</span>
						<span class="field-label">{field.label}</span>
						<div class="toggle" class:active={(fields as any)[field.key]}></div>
					</button>
				{/each}
			</div>
		</div>

		<!-- Advanced Settings -->
		<div class="card">
			<button type="button" class="section-toggle" onclick={() => showAdvanced = !showAdvanced}>
				<div class="flex items-center gap-2">
					<span class="mi">settings</span>
					<h2 class="section-title" style="margin: 0;">Advanced Settings</h2>
				</div>
				<span class="mi">{showAdvanced ? 'expand_less' : 'expand_more'}</span>
			</button>

			{#if showAdvanced}
				<div class="advanced-grid animate-fade-in">
					<div class="input-group">
						<label for="language-input" class="flex items-center gap-1">
							Language
							<span class="mi mi-sm text-muted" title="Language code for Google Maps (e.g., en, de, fr) to fetch localized results.">info</span>
						</label>
						<select id="language-input" class="input" bind:value={language}>
							<option value="en">English</option>
							<option value="de">German</option>
							<option value="fr">French</option>
							<option value="es">Spanish</option>
							<option value="pt">Portuguese</option>
							<option value="ja">Japanese</option>
							<option value="ko">Korean</option>
							<option value="zh">Chinese</option>
							<option value="ar">Arabic</option>
							<option value="bn">Bangla</option>
						</select>
					</div>

					<div class="input-group">
						<label for="depth-input" class="flex items-center gap-1">
							Scroll Depth
							<span class="mi mi-sm text-muted" title="Number of times to scroll down the search results. Depth 10 yields roughly 150-200 places.">info</span>
						</label>
						<input id="depth-input" type="number" class="input" bind:value={maxDepth} min="1" max="50" />
					</div>

					<div class="input-group">
						<label for="zoom-input" class="flex items-center gap-1">
							Zoom Level (0-21)
							<span class="mi mi-sm text-muted" title="Map zoom level to perform the search in. 15 is standard city level.">info</span>
						</label>
						<input id="zoom-input" type="number" class="input" bind:value={zoom} min="0" max="21" />
					</div>

					<div class="input-group">
						<label for="geo-input" class="flex items-center gap-1">
							Geo Coordinates
							<span class="mi mi-sm text-muted" title="Latitude,Longitude format (e.g. 37.7749,-122.4194) forces the search from exactly that location.">info</span>
						</label>
						<input id="geo-input" type="text" class="input" placeholder="37.7749,-122.4194" bind:value={geo} />
					</div>

					<div class="input-group">
						<label for="concurrency-input" class="flex items-center gap-1">
							Concurrency (Workers)
							<span class="mi mi-sm text-muted" title="How many simultaneous browsers to launch. Warning: higher concurrency requires more RAM and CPU.">info</span>
						</label>
						<input id="concurrency-input" type="number" class="input" bind:value={concurrency} min="1" max="10" />
					</div>

					<div class="input-group">
						<label for="max-results-input" class="flex items-center gap-1">
							Max Results (Limit)
							<span class="mi mi-sm text-muted" title="Stop the scraper immediately after hitting this limit. Leave empty for unlimited.">info</span>
						</label>
						<input id="max-results-input" type="number" class="input" placeholder="e.g. 50" bind:value={maxResults} min="1" />
					</div>

					<div class="input-group" style="grid-column: 1 / -1;">
						<label for="webhook-input" class="flex items-center gap-1">
							Webhook URL (Optional)
							<span class="mi mi-sm text-muted" title="A URL to send an automatic HTTP POST payload to when the scraper successfully finishes or fails.">info</span>
						</label>
						<input id="webhook-input" type="url" class="input" placeholder="https://api.example.com/webhook" bind:value={webhookUrl} />
						<span class="text-xs text-secondary mt-1">Receive a JSON POST request when the scrape job completes</span>
					</div>

					<div class="input-group" style="grid-column: 1 / -1;">
						<label for="proxies-input" class="flex items-center gap-1">
							Proxies (one per line)
							<span class="mi mi-sm text-muted" title="List of specific HTTP/SOCKS proxies to route scraping traffic through and avoid IP bans.">info</span>
						</label>
						<textarea
							id="proxies-input"
							class="input"
							placeholder="socks5://user:pass@host:port&#10;http://host:port"
							bind:value={proxies}
							rows="3"
						></textarea>
					</div>
				</div>
			{/if}
		</div>

		<!-- Error -->
		{#if error}
			<div class="error-msg animate-fade-in">
				<span class="mi mi-sm">error</span>
				{error}
			</div>
		{/if}

		<!-- Submit -->
		<button type="submit" class="btn btn-primary submit-btn" disabled={loading}>
			{#if loading}
				<div class="btn-spinner"></div>
				Starting Scrape...
			{:else}
				<span class="mi">rocket_launch</span>
				Start Scraping
			{/if}
		</button>
	</form>
{/if}

<style>
	.page-header {
		margin-bottom: 28px;
	}

	.page-header h1 {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 1.6rem;
		font-weight: 700;
		margin-bottom: 4px;
	}

	.page-header h1 .mi {
		color: var(--accent-2);
	}

	.scrape-form {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 1.05rem;
		font-weight: 600;
		margin-bottom: 12px;
	}

	.section-title .mi {
		color: var(--accent-2);
		font-size: 22px;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 16px;
	}

	/* Presets */
	.presets {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
		margin-bottom: 20px;
	}

	.preset-chip {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 14px;
		border-radius: 20px;
		font-size: 0.8rem;
		font-weight: 500;
		background: var(--bg-input);
		color: var(--text-secondary);
		border: 1px solid var(--border);
		cursor: pointer;
		transition: all var(--transition-fast);
		font-family: inherit;
	}

	.preset-chip:hover {
		background: rgba(108, 92, 231, 0.1);
		color: var(--accent-2);
		border-color: var(--border-accent);
	}

	/* Field toggles grid */
	.fields-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 8px;
	}

	.field-toggle {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 14px;
		border-radius: var(--radius-sm);
		background: var(--bg-input);
		border: 1px solid var(--border);
		cursor: pointer;
		transition: all var(--transition-fast);
		font-family: inherit;
		color: var(--text-secondary);
	}

	.field-toggle:hover {
		border-color: var(--border-hover);
		background: var(--bg-card-hover);
	}

	.field-toggle.active {
		border-color: var(--border-accent);
		background: rgba(108, 92, 231, 0.06);
		color: var(--text-primary);
	}

	.field-toggle.active .mi {
		color: var(--accent-2);
	}

	.field-toggle .mi {
		font-size: 18px;
		color: var(--text-muted);
		transition: color var(--transition-fast);
	}

	.field-label {
		flex: 1;
		font-size: 0.85rem;
		font-weight: 500;
		text-align: left;
	}

	.field-toggle .toggle {
		width: 36px;
		height: 20px;
	}

	.field-toggle .toggle::after {
		width: 14px;
		height: 14px;
	}

	.field-toggle .toggle.active::after {
		left: 18px;
	}

	/* Advanced settings */
	.section-toggle {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		background: none;
		border: none;
		cursor: pointer;
		color: var(--text-primary);
		font-family: inherit;
		padding: 0;
	}

	.advanced-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 16px;
		margin-top: 20px;
		padding-top: 16px;
		border-top: 1px solid var(--border);
	}

	/* Error */
	.error-msg {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		border-radius: var(--radius-sm);
		background: rgba(231, 76, 60, 0.1);
		border: 1px solid rgba(231, 76, 60, 0.2);
		color: #e74c3c;
		font-size: 0.85rem;
	}

	/* Submit button */
	.submit-btn {
		height: 50px;
		font-size: 1rem;
		justify-content: center;
		border-radius: var(--radius-md);
	}

	.btn-spinner {
		width: 18px;
		height: 18px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	/* Live Results */
	.live-results {
		background: var(--bg-card);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		overflow: hidden;
	}

	.live-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 20px 24px;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
		gap: 12px;
	}

	.live-header h2 {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 1.1rem;
		font-weight: 600;
	}

	.live-header h2 .mi {
		color: var(--accent-3);
	}

	.results-table-wrapper {
		overflow-x: auto;
	}

	.title-cell strong {
		color: var(--text-primary);
	}

	.rating {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-weight: 600;
		font-size: 0.9rem;
	}

	.link {
		color: var(--accent-2);
		text-decoration: none;
	}

	.link:hover {
		text-decoration: underline;
	}

	select.input {
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%238888a8' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 12px center;
		padding-right: 36px;
	}

	.demo-banner {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 14px 18px;
		border-radius: var(--radius-md);
		background: rgba(243, 156, 18, 0.08);
		border: 1px solid rgba(243, 156, 18, 0.2);
		color: #f39c12;
		font-size: 0.85rem;
		line-height: 1.5;
		margin-bottom: 20px;
	}
	.demo-banner .mi {
		margin-top: 2px;
		flex-shrink: 0;
	}
	.demo-banner strong {
		color: #f5a623;
	}
	.demo-banner div {
		color: var(--text-secondary);
	}
</style>
