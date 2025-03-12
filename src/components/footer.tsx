import { Image } from './ui/image';

export function Footer() {
	return (
		<footer className='w-full py-12 px-6 bg-neutral-100 border-t border-neutral-200'>
			<div className='max-w-7xl mx-auto'>
				<div className='flex flex-col md:flex-row gap-8'>
					<div className='flex-1'>
						<Image
							src='/placeholder.svg?height=40&width=120'
							alt='Euro Haus'
							width={120}
							height={40}
							className='h-10 w-auto mb-4'
						/>
						<p className='text-neutral-600 mb-4'>
							Specialized European auto service and performance shop dedicated
							to excellence.
						</p>
						<div className='flex gap-3'>
							{['facebook', 'instagram', 'youtube'].map((social) => (
								<a
									key={social}
									href='#'
									className='p-2 rounded-lg bg-neutral-100 shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300'
								>
									<span className='sr-only'>{social}</span>
									<div className='h-5 w-5 text-neutral-700'>
										{social.charAt(0).toUpperCase()}
									</div>
								</a>
							))}
						</div>
					</div>

					{[
						// {
						// 	title: 'Services',
						// 	links: [
						// 		'Performance Tuning',
						// 		'Maintenance',
						// 		'Custom Builds',
						// 		'Detailing',
						// 	],
						// },
						{
							title: 'Company',
							links: ['About Us', 'Our Team', 'Careers', 'Contact Us'],
						},
						{
							title: 'Information',
							links: [
								'FAQ',
								'Shipping & Returns',
								'Privacy Policy',
								'Terms of Service',
							],
						},
					].map((column, index) => (
						<div key={index} className='flex-1'>
							<h3 className='font-semibold text-neutral-800 mb-4'>
								{column.title}
							</h3>
							<ul className='space-y-2'>
								{column.links.map((link) => (
									<li key={link}>
										<a
											href='#'
											className='text-neutral-600 hover:text-neutral-800 transition-colors'
										>
											{link}
										</a>
									</li>
								))}
							</ul>
						</div>
					))}
				</div>

				<div className='mt-12 pt-6 border-t border-neutral-200 text-center text-neutral-600 text-sm'>
					© {new Date().getFullYear()} The Euro Haus. All rights reserved.
				</div>
			</div>
		</footer>
	);
}
