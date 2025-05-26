import * as React from 'react';
import { Search, ShoppingBag, User, Menu } from 'lucide-react';

import { cn } from '~/lib/utils';
import { Image } from './ui/image';
import { Button, buttonVariants } from './ui/button';
import {
	NavigationMenu,
	NavigationMenuContent,
	NavigationMenuItem,
	NavigationMenuLink,
	NavigationMenuList,
	NavigationMenuTrigger,
	navigationMenuTriggerStyle
} from './ui/navigation-menu';
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetHeader,
	SheetTitle,
	SheetTrigger
} from './ui/sheet';
import { Link } from '@tanstack/react-router';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from './ui/accordion';

type NavLink = {
	title: string;
	href: string;
} & (
		| { hasResources?: undefined | false; resources?: NavResource[] }
		| { hasResources: true; resources: NavResource[] }
	);

type NavResource = NavLink & {
	description?: string;
};

const navLinks: NavLink[] = [
	{
		title: 'Home',
		href: '/',
	},
	{
		title: 'About',
		href: '/about',
	},
	{
		title: 'Catalog',
		href: '/catalog',
	},
]

export function Navbar() {
	return (
		<header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
			<div className="px-8 flex h-16 items-center">
				<MainNav />
				<div className="ml-auto flex items-center space-x-4">
					<a href="/search" className="text-muted-foreground hover:text-foreground">
						<span className="sr-only">Search</span>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="24"
							height="24"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							strokeWidth="2"
							strokeLinecap="round"
							strokeLinejoin="round"
							className="h-5 w-5"
						>
							<circle cx="11" cy="11" r="8" />
							<path d="m21 21-4.3-4.3" />
						</svg>
					</a>
					<a href="/account" className="text-muted-foreground hover:text-foreground">
						<span className="sr-only">Account</span>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="24"
							height="24"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							strokeWidth="2"
							strokeLinecap="round"
							strokeLinejoin="round"
							className="h-5 w-5"
						>
							<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
							<circle cx="12" cy="7" r="4" />
						</svg>
					</a>
					<a href="/cart" className="text-muted-foreground hover:text-foreground">
						<span className="sr-only">Cart</span>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="24"
							height="24"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							strokeWidth="2"
							strokeLinecap="round"
							strokeLinejoin="round"
							className="h-5 w-5"
						>
							<circle cx="8" cy="21" r="1" />
							<circle cx="19" cy="21" r="1" />
							<path d="M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12" />
						</svg>
					</a>
				</div>
			</div>
		</header>
	)
}

export function MainNav() {
	return (
		<div className='flex items-center gap-6 md:gap-10'>
			<Link to='/' className='hidden items-center space-x-2 md:flex'>
				{/* <span className='hidden font-bold sm:inline-block'>EURO HAUS</span> */}
				<Image src="/eurohaus-logo.png" alt="Euro Haus Logo" width={100} height={100} />
			</Link>
			<NavigationMenu className='flex'>
				<Sheet>
					<SheetTrigger asChild>
						<Button variant='ghost' className='px-0 text-base hover:bg-transparent focus-visible:bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0 md:hidden'>
							<Menu className='h-5 w-5' />
							<span className='sr-only'>Toggle Menu</span>
						</Button>
					</SheetTrigger>
					<SheetContent side='left' className='pr-0'>
						<SheetHeader>
							<SheetTitle>EURO HAUS</SheetTitle>
							<SheetDescription>Come for the cars, stay for the people!</SheetDescription>
						</SheetHeader>
						<nav className='grid gap-6 text-lg font-medium'>
							<Accordion type="single" collapsible className='w-full flex-col flex'>
								{navLinks.map((link) => (
									link.hasResources ? (
										<AccordionItem value={link.title} key={link.title} className='text-start'>
											<AccordionTrigger className={`${navigationMenuTriggerStyle()} items-center justify-between`}>
												{link.title}
											</AccordionTrigger>
											<AccordionContent>
												<div className="flex flex-col space-y-2 pl-4">
													{link.resources?.map((resource) => (
														<NavigationMenuLink
															key={resource.title}
															href={resource.href}
															className={navigationMenuTriggerStyle()}
														>
															{resource.title}
														</NavigationMenuLink>
													))}
												</div>
											</AccordionContent>
										</AccordionItem>
									) : (
										<NavigationMenuLink
											key={link.title}
											href={link.href}
											className={`${navigationMenuTriggerStyle()} self-start`}
										>
											{link.title}
										</NavigationMenuLink>
									)
								))}
							</Accordion>
						</nav>
					</SheetContent>
				</Sheet>
				<NavigationMenuList className='hidden md:flex'>
					{navLinks.map((link) => (
						<NavigationMenuItem key={link.title}>
							{link.hasResources ? (
								<>
									<NavigationMenuTrigger>{link.title}</NavigationMenuTrigger>
									<NavigationMenuContent className='bg-card text-card-foreground'>
										<ul className='grid w-[400px] gap-3 p-4 md:w-[500px] md:grid-cols-2 lg:w-[600px]'>
											{link.resources?.map((resource) => (
												<ListItem
													key={resource.title}
													title={resource.title}
													href={resource.href}
												>
													{resource.description}
												</ListItem>
											))}
										</ul>
									</NavigationMenuContent>
								</>
							) : (
								<NavigationMenuLink href={link.href} className={buttonVariants({ variant: 'ghost' })}>
									{link.title}
								</NavigationMenuLink>
							)}
						</NavigationMenuItem>
					))}
				</NavigationMenuList>
			</NavigationMenu>
		</div>
	);
}

const ListItem = React.forwardRef<React.ComponentRef<"a">, React.ComponentPropsWithoutRef<"a">>(
	({ className, title, children, ...props }, ref) => {
		return (
			<li>
				<NavigationMenuLink asChild>
					<a
						ref={ref}
						className={cn(
							"block select-none space-y-1 rounded-md p-3 leading-none no-underline outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground",
							className,
						)}
						{...props}
					>
						<div className="text-sm font-medium leading-none">{title}</div>
						<p className="line-clamp-2 text-sm leading-snug text-muted-foreground">{children}</p>
					</a>
				</NavigationMenuLink>
			</li>
		)
	},
)

ListItem.displayName = "ListItem"
