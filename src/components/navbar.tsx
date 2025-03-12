import { Search, ShoppingBag, User } from 'lucide-react';
import { Image } from './ui/image';

export function Navbar() {
	return (
		// <header className="w-full px-6 py-4 sticky top-0 "></header>
		<header className='w-full px-6 py-4 bg-secondary sticky top-0 z-50'>
			<div className='max-w-7xl mx-auto flex items-center justify-between'>
				<div className='flex items-center gap-12'>
					<a href='/' className='relative'>
						<div className='p-3 rounded-xl bg-secondary shadow-neumorph'>
							<Image
								src='/placeholder.svg?height=40&width=120'
								alt='Euro Haus'
								width={120}
								height={40}
								className='h-10 w-auto'
							/>
						</div>
					</a>

					<nav className='hidden md:flex items-center gap-8'>
						<a
							href='/'
							className='relative py-2 px-4 rounded-xl bg-secondary shadow-neumorph-active font-medium'
						>
							Home
						</a>
						<a
							href='/catalog'
							className='relative py-2 px-4 rounded-xl bg-secondary shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300 font-medium'
						>
							Catalog
						</a>
						<a
							href='/contact'
							className='relative py-2 px-4 rounded-xl bg-secondary shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300 font-medium'
						>
							Contact
						</a>
					</nav>
				</div>

				<div className='flex items-center gap-4'>
					<button className='p-3 rounded-xl bg-secondary shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300'>
						<Search className='h-5 w-5' />
						<span className='sr-only'>Search</span>
					</button>
					<button className='p-3 rounded-xl bg-secondary shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300'>
						<User className='h-5 w-5' />
						<span className='sr-only'>Account</span>
					</button>
					<button className='p-3 rounded-xl bg-secondary shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300'>
						<ShoppingBag className='h-5 w-5' />
						<span className='sr-only'>Cart</span>
					</button>
				</div>
			</div>
		</header>
	);
}
