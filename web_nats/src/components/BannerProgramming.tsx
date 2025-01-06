// src/components/BannerProgramming.tsx
import React, { useState } from 'react';
import { handleBannerClick, handleMouseEnter } from './banners';

interface BannerProgrammingProps {
  titleSlogan: string;
  subtitleSlogan: string;
  imageUrl: string;
}

const BannerProgramming: React.FC<BannerProgrammingProps> = ({ imageUrl, titleSlogan, subtitleSlogan}) => {
  // Estado para mostrar u ocultar el resumen
  const [isHovered, setIsHovered] = useState(false);

  return (
    <div
      className="chat-banner-programming"
      onClick={() => handleBannerClick('Programación en la Nube')}
      onMouseEnter={() => {
        handleMouseEnter('Programación en la Nube');
        setIsHovered(true); // Mostrar el resumen al pasar el mouse
      }}
      onMouseLeave={() => setIsHovered(false)} // Ocultar el resumen cuando el mouse sale
    >
      <div className="banner-content">
       
        {/* Mostrar la imagen del banner */}
        <img src={imageUrl} alt="Banner de Programación en la Nube" className="banner-image" />
        
        {/* Agregar el eslogan debajo de la imagen */}
        <div className="banner-slogan">
          <h2>{titleSlogan}</h2>
          <p>{subtitleSlogan}</p>
        </div>

        {/* Si el mouse está encima, mostrar el resumen */}
       {/* Si el mouse está encima, mostrar el resumen */}
       {isHovered && (
        <div className="chat-icon-banner">
          <div className="chat-tech-icons">
            <img src="https://miro.medium.com/v2/resize:fit:740/1*rxDdNJHiz1R38J_JEz23Zw.jpeg" alt="Go" className="chat-icon" />
            <img src="https://upload.wikimedia.org/wikipedia/commons/d/d9/Node.js_logo.svg" alt="Node.js" className="chat-icon" />
            <img src="https://www.typescriptlang.org/images/branding/two-longform.svg" alt="TypeScript" className="chat-icon" />
            <img src="https://upload.wikimedia.org/wikipedia/commons/4/47/React.svg" alt="React" className="chat-icon" />
            <img src="/images/concepts.png" alt="concepts" className="chat-icon" />
          </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default BannerProgramming;