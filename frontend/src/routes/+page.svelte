<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import HealthIcon from '#lib/HealthIcon.svelte';
	import HeroThree from '#lib/HeroThree.svelte';
	import LandingNav from '#lib/LandingNav.svelte';
	import './landing.css';
	import './landing-sections.css';

	let selectedRole = $state(0);
	let selectedPermission = $state(0);
	const permissionStages = [
		{
			label: 'Pasien',
			icon: 'care' as const,
			title: 'Anda menentukan siapa yang menerima.',
			text: 'Dokter memerlukan hubungan perawatan atau izin yang sah. Laboratorium dan apotek menerima penugasan yang Anda setujui.'
		},
		{
			label: 'Persetujuan',
			icon: 'shield' as const,
			title: 'Bagikan sesuai kebutuhan perawatan.',
			text: 'Penerima, jenis data, dan masa akses mengikuti izin yang diberikan. Persetujuan pemeriksaan atau resep tidak membuka seluruh riwayat Anda.'
		},
		{
			label: 'Penyedia layanan',
			icon: 'lab' as const,
			title: 'Setiap pihak melanjutkan bagiannya.',
			text: 'Laboratorium mengerjakan pemeriksaan yang ditugaskan. Apotek menyiapkan resep yang dibagikan. Akses mengikuti masa episode, penugasan, atau izin yang berlaku.'
		}
	];
	let ready = $state(false);
	let scrollProgress = $state(0);

	function updateScrollProgress() {
		const scrollTop = window.scrollY || document.documentElement.scrollTop;
		const docHeight = document.documentElement.scrollHeight - window.innerHeight;
		scrollProgress = docHeight > 0 ? Math.min(1, Math.max(0, scrollTop / docHeight)) : 0;
	}

	onMount(() => {
		ready = true;
		updateScrollProgress();
		const observer = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						entry.target.classList.add('in-view');
						observer.unobserve(entry.target);
					}
				});
			},
			{ threshold: 0.08, rootMargin: '0px 0px -40px 0px' }
		);
		document.querySelectorAll('.scroll-reveal').forEach((el) => {
			if (el.getBoundingClientRect().top > window.innerHeight) el.classList.add('reveal-pending');
			observer.observe(el);
		});
		const steps = [...document.querySelectorAll('.flow-steps li')];
		let stepFrame = 0;
		function updateStep() {
			stepFrame = 0;
			const bounds = steps.map((step) => step.getBoundingClientRect());
			if (
				!bounds.length ||
				bounds[0].top > window.innerHeight ||
				bounds[bounds.length - 1].bottom < 0
			)
				return;
			const readingLine = window.innerHeight * 0.33;
			const activeIndex = bounds.reduce(
				(best, rect, index) =>
					Math.abs(rect.top - readingLine) < Math.abs(bounds[best].top - readingLine)
						? index
						: best,
				0
			);
			steps.forEach((step, index) => {
				step.classList.toggle('is-current', index === activeIndex);
				step.classList.toggle('is-complete', index < activeIndex);
				if (index === activeIndex) step.setAttribute('aria-current', 'step');
				else step.removeAttribute('aria-current');
			});
		}
		function scheduleStep() {
			if (!stepFrame) stepFrame = requestAnimationFrame(updateStep);
		}
		window.addEventListener('scroll', scheduleStep, { passive: true });
		window.addEventListener('resize', scheduleStep);
		updateStep();
		return () => {
			observer.disconnect();
			cancelAnimationFrame(stepFrame);
			window.removeEventListener('scroll', scheduleStep);
			window.removeEventListener('resize', scheduleStep);
		};
	});

	const audiences = [
		{
			label: 'Pasien',
			title: 'Pahami apa yang sedang berjalan.',
			description:
				'Lihat hubungan perawatan, status pemeriksaan, resep, dan riwayat kejadian dari ruang pasien Anda.',
			items: [
				'Ikuti permintaan pemeriksaan dan statusnya di laboratorium.',
				'Baca nama obat, dosis, serta petunjuk yang tercantum pada resep.',
				'Lihat perkembangan perawatan dalam timeline yang tersusun.'
			],
			access:
				'Pasien melihat data perawatannya sendiri. Berbagi dengan penyedia lain mengikuti persetujuan dan cakupan akses yang berlaku.'
		},
		{
			label: 'Dokter',
			title: 'Lanjutkan perawatan dengan konteks yang jelas.',
			description:
				'Alur dokter menghubungkan pasien, permintaan pemeriksaan, hasil, dan resep melalui episode perawatan.',
			items: [
				'Mulai hubungan perawatan melalui undangan kepada pasien.',
				'Ajukan pemeriksaan dan baca hasil sesuai hak akses.',
				'Terbitkan resep berisi obat, dosis, jumlah, dan aturan pakai.'
			],
			access:
				'Hubungan dalam satu klinik tidak otomatis memberikan akses ke seluruh pasien. Akses klinis mengikuti episode aktif atau izin lain yang sah.'
		},
		{
			label: 'Laboratorium',
			title: 'Dari permintaan menjadi hasil pemeriksaan.',
			description:
				'Laboratorium menerima penugasan pemeriksaan setelah pasien menyetujui pengiriman permintaan tersebut.',
			items: [
				'Terima permintaan yang ditujukan ke laboratorium Anda.',
				'Perbarui proses dari diterima menjadi sedang diperiksa.',
				'Kirim hasil terstruktur untuk perjalanan perawatan terkait.'
			],
			access:
				'Penugasan lab berfokus pada pemeriksaan yang disetujui. Penugasan tersebut tidak membuka seluruh riwayat klinis pasien.'
		},
		{
			label: 'Apotek',
			title: 'Siapkan resep yang ditujukan kepada Anda.',
			description:
				'Pasien memilih apotek dan menyetujui pengiriman resep. Apotek tujuan melanjutkan proses penyiapan.',
			items: [
				'Terima resep yang sudah dibagikan ke apotek Anda.',
				'Ikuti tahap diterima, disiapkan, siap diambil, hingga selesai.',
				'Gunakan detail obat dan aturan pakai yang diterbitkan dokter.'
			],
			access:
				'Akses apotek dibatasi pada resep yang ditugaskan selama prosesnya aktif. Data pemeriksaan lain tidak ikut dibuka.'
		}
	];

	let audience = $derived(audiences[selectedRole]);

	const questions = [
		{
			question: 'Apakah semua dokter bisa melihat data saya?',
			answer:
				'Tidak. Dokter memerlukan hubungan perawatan aktif atau izin lain yang sah untuk data terkait. Dokter lain, termasuk rekan satu klinik, tidak otomatis mendapat akses.'
		},
		{
			question: 'Apakah riwayat perawatan sama dengan riwayat akses?',
			answer:
				'Berbeda. Timeline perawatan mencatat undangan, proses lab, resep, dan persetujuan. Riwayat akses mencatat aktivitas melihat data. Keduanya tersedia di menu terpisah pada ruang pasien.'
		},
		{
			question: 'Bagaimana resep sampai ke apotek?',
			answer:
				'Dokter menerbitkan resep, lalu pasien memilih apotek dan menyetujui pengirimannya dari menu Resep & apotek. Petugas apotek menerima dan menyiapkan resep melalui ruang apotek. Pasien bisa mengikuti status penyiapannya.'
		},
		{
			question: 'Apakah Hygeia menentukan obat atau diagnosis?',
			answer:
				'Tidak. Informasi obat berasal dari resep dokter. Hygeia menghubungkan informasi perawatan dan menampilkan perkembangannya; aplikasi tidak membuat keputusan diagnosis atau resep otomatis.'
		},
		{
			question: 'Apa yang sudah bisa saya coba sekarang?',
			answer:
				'Pasien dapat menyetujui perawatan dan pemeriksaan, memilih apotek, mengatur izin, serta membagikan CarePass. Dokter dapat mengundang pasien, meminta pemeriksaan, menerbitkan resep, dan membaca data sesuai izin.'
		}
	];

	const features = [
		{
			icon: 'care' as const,
			title: 'Perawatan yang terhubung',
			text: 'Ikuti hubungan perawatan dengan dokter dan perkembangan setiap perjalanan Anda.'
		},
		{
			icon: 'lab' as const,
			title: 'Pemeriksaan lebih mudah diikuti',
			text: 'Lihat permintaan pemeriksaan dan status prosesnya di laboratorium.'
		},
		{
			icon: 'medicine' as const,
			title: 'Resep dalam satu tempat',
			text: 'Baca obat yang diresepkan, aturan pakai, dan status penyiapan oleh apotek.'
		},
		{
			icon: 'shield' as const,
			title: 'Akses dengan persetujuan',
			text: 'Akses data mengikuti izin pasien dan hubungan perawatan yang berlaku.'
		}
	];
</script>

<svelte:head>
	<link rel="preload" as="image" href="/hero-clinic-v2.png" />
	<title>Hygeia · Perawatan terhubung, kendali di tangan Anda</title>
	<meta
		name="description"
		content="Ikuti perjalanan perawatan, pemeriksaan, dan resep Anda bersama Hygeia. Platform pertukaran data kesehatan dengan persetujuan pasien."
	/>
</svelte:head>

<svelte:window onscroll={updateScrollProgress} />

<div class="landing">
	<div
		class="scroll-progress-bar"
		style:transform="scaleX({scrollProgress})"
		role="progressbar"
		aria-valuenow={Math.round(scrollProgress * 100)}
		aria-valuemin="0"
		aria-valuemax="100"
		aria-label="Progres baca beranda"
	></div>
	<a class="landing-skip" href="#utama">Lewati ke konten</a>
	<LandingNav onSelectRole={(index) => (selectedRole = index)} />

	<main id="utama">
		<!-- HERO SECTION -->
		<HeroThree onSelectRole={(index) => (selectedRole = index)} />

		<!-- FEATURES SECTION (Scroll Reveal) -->
		<section class="landing-features scroll-reveal" id="fitur" aria-labelledby="features-title">
			<div class="features-intro">
				<p class="landing-kicker">SETIAP LANGKAH, LEBIH JELAS</p>
				<h2 id="features-title">Dari pemeriksaan<br />sampai resep.</h2>
				<p>
					Permintaan lab sedang diproses? Resep sudah siap diambil? Dashboard memperlihatkan status
					dari layanan yang menangani Anda.
				</p>
			</div>
			<div class="feature-list">
				{#each features as feature (feature.title)}
					<article>
						<span class="feature-icon"><HealthIcon name={feature.icon} /></span>
						<div>
							<h3>{feature.title}</h3>
							<p>{feature.text}</p>
						</div>
					</article>
				{/each}
			</div>
		</section>

		<!-- AUDIENCES / ROLES SECTION (Scroll Reveal) -->
		<section
			class="landing-audiences scroll-reveal"
			id="untuk-siapa"
			aria-labelledby="audiences-title"
		>
			<div class="audiences-heading">
				<p class="landing-kicker">RUANG UNTUK SETIAP PERAN</p>
				<h2 id="audiences-title">Empat peran.<br />Satu perjalanan perawatan.</h2>
				<p>
					Kebutuhan setiap pihak berbeda. Informasi yang dibagikan mengikuti peran dan hubungan
					perawatannya.
				</p>
			</div>

			<div class="audience-explorer">
				<div class="audience-options" aria-label="Pilih peran untuk mengetahui alurnya">
					{#each audiences as item, index (item.label)}
						<button
							disabled={!ready}
							aria-pressed={selectedRole === index}
							onclick={() => (selectedRole = index)}
						>
							{item.label}<span aria-hidden="true">→</span>
						</button>
					{/each}
				</div>

				<div class="audience-detail" aria-live="polite">
					{#key selectedRole}
						<div class="audience-detail-fade">
							<h3>{audience.title}</h3>
							<p>{audience.description}</p>
							<ul>
								{#each audience.items as item (item)}
									<li>{item}</li>
								{/each}
							</ul>
							<div class="audience-access">
								<HealthIcon name="shield" />
								<p>{audience.access}</p>
							</div>
						</div>
					{/key}
				</div>
			</div>
		</section>

		<!-- FLOW SECTION (Scroll Reveal) -->
		<section class="landing-flow scroll-reveal" id="cara-kerja" aria-labelledby="flow-title">
			<div class="flow-heading">
				<p class="landing-kicker">DARI UNDANGAN HINGGA RESEP</p>
				<h2 id="flow-title">Perawatan dimulai<br />dengan persetujuan Anda.</h2>
				<p>
					Hubungan dengan dokter menjadi dasar perjalanan perawatan. Akses data mengikuti izin yang
					berlaku.
				</p>
			</div>
			<ol class="flow-steps">
				<li>
					<h3>Dokter mengundang</h3>
					<p>Dokter mengirim undangan untuk memulai hubungan perawatan dengan pasien.</p>
				</li>
				<li>
					<h3>Pasien menyetujui</h3>
					<p>Pasien meninjau undangan dan menentukan apakah perawatan dapat dimulai.</p>
				</li>
				<li>
					<h3>Pemeriksaan dikirim ke lab</h3>
					<p>
						Dokter mengajukan pemeriksaan. Setelah pasien menyetujui pengiriman, laboratorium tujuan
						menerima permintaan dan memprosesnya.
					</p>
				</li>
				<li>
					<h3>Hasil menjadi bagian dari perawatan</h3>
					<p>
						Laboratorium mengirim hasil terstruktur. Dokter dapat meninjaunya sesuai hak akses untuk
						melanjutkan penilaian klinis.
					</p>
				</li>
				<li>
					<h3>Resep dilanjutkan ke apotek pilihan</h3>
					<p>
						Jika dokter menerbitkan resep, pasien memilih apotek dan menyetujui pengiriman. Status
						penyiapan dapat diikuti sampai resep selesai.
					</p>
				</li>
			</ol>
		</section>

		<!-- PERMISSION & SECURITY SECTION (Scroll Reveal) -->
		<section
			class="landing-permission scroll-reveal"
			id="izin-data"
			aria-labelledby="permission-title"
		>
			<div class="permission-intro">
				<p class="landing-kicker">IZIN DATA, DI TANGAN ANDA</p>
				<h2 id="permission-title">Berbagi seperlunya.<br />Dengan tujuan yang jelas.</h2>
				<p>Ikuti alurnya untuk memahami bagaimana informasi perawatan dibagikan.</p>
			</div>
			<div class="permission-path" aria-label="Jelajahi alur izin data">
				{#each permissionStages as stage, index (stage.label)}
					<button
						aria-pressed={selectedPermission === index}
						aria-controls="permission-explanation"
						onclick={() => (selectedPermission = index)}
					>
						<span class="permission-node"><HealthIcon name={stage.icon} /></span>
						<span class="permission-stage-number">0{index + 1}</span>
						<span>{stage.label}</span>
					</button>
				{/each}
			</div>
			<div class="permission-explanation" id="permission-explanation" aria-live="polite">
				{#key selectedPermission}
					<div class="audience-detail-fade">
						<h3>{permissionStages[selectedPermission].title}</h3>
						<p>{permissionStages[selectedPermission].text}</p>
					</div>
				{/key}
			</div>
			<p class="permission-footnote">
				<HealthIcon name="shield" /> Timeline perawatan dan riwayat akses dapat dilihat sebagai dua catatan
				terpisah.
			</p>
		</section>

		<!-- FAQ SECTION (Scroll Reveal) -->
		<section class="landing-faq scroll-reveal" id="pertanyaan" aria-labelledby="faq-title">
			<div class="faq-intro">
				<p class="landing-kicker">PERTANYAAN YANG SERING DITANYAKAN</p>
				<h2 id="faq-title">Mari mengenal Hygeia.</h2>
				<p>Tentang akses data, alur resep, dan layanan yang tersedia.</p>
			</div>
			<div class="faq-list">
				{#each questions as item (item.question)}
					<details>
						<summary>{item.question}<span aria-hidden="true">+</span></summary>
						<p>{item.answer}</p>
					</details>
				{/each}
			</div>
		</section>

		<!-- ABOUT & ACTION SECTION (Scroll Reveal) -->
		<section class="landing-about scroll-reveal" id="tentang" aria-labelledby="about-title">
			<!-- Centered actions adapted from sv-blocks call-to-action-one. -->
			<div class="closing-card">
				<p class="closing-eyebrow">KENALI PERAWATAN ANDA LEBIH DEKAT</p>
				<h2 id="about-title">Satu ruang untuk mengikuti<br />perjalanan perawatan Anda.</h2>
				<p>
					Hygeia menghubungkan pasien, dokter, laboratorium, dan apotek. Ikuti perkembangannya,
					dengan persetujuan tetap di tangan Anda.
				</p>
				<div class="closing-actions">
					<a class="closing-primary" href={resolve('/register')}
						>Mulai dengan Hygeia <span aria-hidden="true">↗</span></a
					>
					<a class="closing-secondary" href={resolve('/login')}>Masuk ke akun</a>
				</div>
				<span class="closing-note">Perawatan terhubung. Kendali di tangan Anda.</span>
			</div>
		</section>
	</main>

	<!-- FOOTER -->
	<footer class="landing-footer">
		<a class="brand" href={resolve('/')}>
			<img src="/logo-light.png" width="32" height="32" alt="" /><span>Hygeia</span>
		</a>
		<p>Perawatan terhubung. Kendali di tangan Anda.</p>
		<nav aria-label="Informasi tambahan">
			<a href="#untuk-siapa">Untuk siapa</a>
			<a href="#izin-data">Izin data</a>
			<a href="#pertanyaan">Pertanyaan</a>
			<a href={resolve('/login')}>Masuk</a>
		</nav>
	</footer>
</div>
